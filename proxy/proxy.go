// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Proxy is used to accept flows and relay them to multiple targets

package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/stats"
	"github.com/dmabry/flowgre/utils"
	"golang.org/x/sync/errgroup"
)

const (
	udpMaxBufferSize = 65507
	bufferSize       = 1024
	// sourcePortMin/Max define the range for random source ports
	sourcePortMin = 10000
	sourcePortMax = 15000
	// maxTargets is the hard limit on number of proxy targets
	maxTargets = 10
)

// Worker is the goroutine used to create workers
func worker(id int, ctx context.Context, server string, port int, wg *sync.WaitGroup, workerChan <-chan []byte) {
	defer wg.Done()
	if err := runWorker(id, ctx, server, port, workerChan); err != nil {
		log.Printf("Worker [%2d] error: %v", id, err)
	}
}

func runWorker(id int, ctx context.Context, server string, port int, workerChan <-chan []byte) error {
	// Sent limiter to given delay
	// Configure connection to use.  It looks like a listener, but it will be used to send packet.  Allows me to set the source port
	srcPort := utils.RandomNum(sourcePortMin, sourcePortMax)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		return fmt.Errorf("listen on source port %d: %w", srcPort, err)
	}
	defer conn.Close()
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(server)
	log.Printf("Worker [%2d] Sending flows at %s:%d\n",
		id, server, port)
	//Infinite loop to keep slinging until we receive context done.
	for {
		select {
		case <-ctx.Done(): //Caught the signal to be done.... time to wrap it up
			log.Printf("Worker [%2d] exiting due to signal\n", id)
			return nil
		case payload, ok := <-workerChan:
			if !ok {
				return nil
			}
			// length := len(payload)
			//log.Printf("Worker [%2d] sending packet to %s:%d with length: %d\n", id, server, port, length)
			// send packet here.
			_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, payload, false)
			if err != nil {
				return fmt.Errorf("send packet: %w", err)
			}
		}
	}
}

// replicator is used to take payloads off the dataChan and pass it to each worker's channel for sending
func replicator(ctx context.Context, wg *sync.WaitGroup, dataChan <-chan []byte, targets []chan []byte, verbose bool) {
	defer wg.Done()
	_ = runReplicator(ctx, dataChan, targets, verbose)
}

func runReplicator(ctx context.Context, dataChan <-chan []byte, targets []chan []byte, verbose bool) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("Replicator exiting due to signal")
			return nil
		case payload, ok := <-dataChan:
			if !ok {
				return nil
			}
			for _, target := range targets {
				select {
				case target <- payload:
					// sent successfully
				case <-ctx.Done():
					log.Println("Replicator context cancelled during send")
					return nil
				default:
					// Channel full, drop packet to avoid deadlock
					if verbose {
						log.Printf("Replicator: dropped packet (target channel full)")
					}
				}
			}
		}
	}
}

// proxyListener is used to pull packets off the wire and put the byte payload on the data chan
func proxyListener(ctx context.Context, wg *sync.WaitGroup, ip string, port int, proxyChan chan<- []byte, verbose bool) {
	defer wg.Done()
	if err := runProxyListener(ctx, ip, port, proxyChan, verbose); err != nil {
		log.Printf("Proxy listener error: %v", err)
	}
}

func runProxyListener(ctx context.Context, ip string, port int, proxyChan chan<- []byte, verbose bool) error {
	// Create UDP listener and setup db to catch files
	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		return fmt.Errorf("listen on %s:%d: %w", ip, port, err)
	}
	log.Printf("Listening on %s:%d", ip, port)
	defer conn.Close()
	stopCancelWakeup := context.AfterFunc(ctx, func() {
		_ = conn.SetReadDeadline(time.Now())
	})
	defer stopCancelWakeup()
	// Start the loop and check context for done, otherwise listen for packets
	for {
		select {
		case <-ctx.Done():
			log.Println("Proxy Listener exiting due to signal")
			return nil
		default:
			payload := make([]byte, udpMaxBufferSize)
			timeout := time.Now().Add(5 * time.Second)
			err := conn.SetReadDeadline(timeout)
			if err != nil {
				return fmt.Errorf("set read deadline: %w", err)
			}
			length, fromIP, err := conn.ReadFromUDP(payload)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					continue
				}
				return fmt.Errorf("read UDP packet: %w", err)
			}
			payload = payload[:length]
			if verbose {
				log.Printf("Packet Received from %s with size of %d", fromIP.String(), length)
			}
			// Send payload to the proxyChan channel
			select {
			case proxyChan <- payload:
			case <-ctx.Done():
				return nil
			default:
				if verbose {
					log.Printf("proxyListener: dropped packet (proxyChan full)")
				}
			}
		}
	}
}

// statsPrinter prints out the status every 10 seconds.
func statsPrinter(ctx context.Context, wg *sync.WaitGroup, rStats *stats.RecordStat) {
	defer wg.Done()
	_ = runStatsPrinter(ctx, rStats)
}

func runStatsPrinter(ctx context.Context, rStats *stats.RecordStat) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			//log.Println("statsPrinter exiting due to signal")
			return nil
		case <-ticker.C:
			log.Printf("Netflow v9 Packets: %d Ignored Packets: %d ",
				rStats.LoadValid(), rStats.LoadInvalid())
		}
	}
}

// parseNetflow validates that the payload is valid NetFlow v9 or IPFIX v10 and forwards it.
func parseNetflow(ctx context.Context, wg *sync.WaitGroup, proxyChan <-chan []byte, dataChan chan<- []byte, rStats *stats.RecordStat, verbose bool) {
	defer wg.Done()
	_ = runParseNetflow(ctx, proxyChan, dataChan, rStats, verbose)
}

func runParseNetflow(ctx context.Context, proxyChan <-chan []byte, dataChan chan<- []byte, rStats *stats.RecordStat, verbose bool) error {

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Flow parser exiting due to signal")
			return nil
		case payload, ok := <-proxyChan:
			if !ok {
				return nil
			}
			ok, err := netflow.IsValidNetFlow(payload, 9)
			if err != nil {
				// Try IPFIX
				ok, err = ipfix.IsValidIPFIX(payload)
				if err != nil {
					if verbose {
						log.Printf("Skipping packet due to issue parsing: %v", err)
					}
				}
			}
			if ok {
				rStats.IncrValid()
				select {
				case dataChan <- payload:
				case <-ctx.Done():
					log.Println("Flow parser context cancelled during send")
					return nil
				default:
					if verbose {
						log.Printf("Flow parser: dropped packet (dataChan full)")
					}
				}
			} else {
				rStats.IncrInvalid()
			}
		case <-ticker.C:
			log.Printf("Flow Packets: %d Ignored Packets: %d",
				rStats.LoadValid(), rStats.LoadInvalid())
		}
	}
}

// Run starts the proxy, accepting flows and relaying them to multiple targets.
// It sets up OS signal handling (SIGINT/SIGTERM) for clean shutdown.
func Run(ip string, port int, verbose bool, targets []string) {
	mgr := lifecycle.New()
	defer mgr.Cancel()

	// Setup signal handling BEFORE starting goroutines to avoid missed signals
	_ = mgr.SetupSignalHandler()

	if err := RunCtx(mgr.Context(), ip, port, verbose, targets); err != nil {
		log.Printf("Proxy error: %v", err)
	}
	mgr.Wait()
}

// RunCtx starts the proxy with an externally managed context and propagates
// startup and runtime failures from every pipeline component.
func RunCtx(ctx context.Context, ip string, port int, verbose bool, targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}
	if len(targets) > maxTargets {
		return fmt.Errorf("can't have more than %d targets", maxTargets)
	}

	proxyChan := make(chan []byte, bufferSize)
	dataChan := make(chan []byte, bufferSize)
	rStats := stats.RecordStat{
		ValidCount:   0,
		InvalidCount: 0,
	}
	// Create dedicated channel per target <= maxTargets
	workers := len(targets)
	workerChans := make([]chan []byte, workers)
	type parsedTarget struct {
		host string
		port int
	}
	parsedTargets := make([]parsedTarget, workers)
	for w := range workers {
		workerChan := make(chan []byte, bufferSize)
		workerChans[w] = workerChan
		target := targets[w]
		targetIP, targetPort, err := net.SplitHostPort(target)
		if err != nil {
			return fmt.Errorf("parse target %q: %w", target, err)
		}
		targetPortInt, err := strconv.Atoi(targetPort)
		if err != nil {
			return fmt.Errorf("parse target %q port: %w", target, err)
		}
		if targetPortInt < 1 || targetPortInt > 65535 {
			return fmt.Errorf("target %q port %d out of range 1-65535", target, targetPortInt)
		}
		if net.ParseIP(targetIP) == nil {
			return fmt.Errorf("target %q has invalid IP address %q", target, targetIP)
		}
		parsedTargets[w] = parsedTarget{host: targetIP, port: targetPortInt}
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for w, target := range parsedTargets {
		id := w + 1
		workerChan := workerChans[w]
		eg.Go(func() error { return runWorker(id, egCtx, target.host, target.port, workerChan) })
	}
	eg.Go(func() error { return runStatsPrinter(egCtx, &rStats) })
	eg.Go(func() error { return runParseNetflow(egCtx, proxyChan, dataChan, &rStats, verbose) })
	eg.Go(func() error { return runReplicator(egCtx, dataChan, workerChans, verbose) })
	eg.Go(func() error { return runProxyListener(egCtx, ip, port, proxyChan, verbose) })

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("proxy: %w", err)
	}
	return nil
}
