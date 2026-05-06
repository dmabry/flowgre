// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Proxy is used to accept flows and relay them to multiple targets

package proxy

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/utils"
)

const udpMaxBufferSize = 65507
const bufferSize = 1024

// Worker is the goroutine used to create workers
func worker(id int, ctx context.Context, server string, port int, wg *sync.WaitGroup, workerChan <-chan []byte) {
	defer wg.Done()
	// Sent limiter to given delay
	// Configure connection to use.  It looks like a listener, but it will be used to send packet.  Allows me to set the source port
	srcPort := utils.RandomNum(10000, 15000)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Printf("Listen failed: %v\n", err)
		return
	}
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(server)
	log.Printf("Worker [%2d] Sending flows at %s:%d\n",
		id, server, port)
	//Infinite loop to keep slinging until we receive context done.
	for {
		select {
		case <-ctx.Done(): //Caught the signal to be done.... time to wrap it up
			log.Printf("Worker [%2d] exiting due to signal\n", id)
			return
		case payload := <-workerChan:
			// length := len(payload)
			//log.Printf("Worker [%2d] sending packet to %s:%d with length: %d\n", id, server, port, length)
			// send packet here.
			var buf bytes.Buffer
			_, err := buf.Write(payload)
			if err != nil {
				log.Printf("Worker [%2d] Issue writing data: %v\n", id, err)
			}
			_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			if err != nil {
				log.Printf("Worker [%2d] Issue sending packet: %v\n", id, err)
				return
			}
		}
	}
}

// replicator is used to take payloads off the dataChan and pass it to each worker's channel for sending
func replicator(ctx context.Context, wg *sync.WaitGroup, dataChan <-chan []byte, targets []chan []byte, verbose bool) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Println("Replicator exiting due to signal")
			return
		case payload := <-dataChan:
			for _, target := range targets {
				select {
				case target <- payload:
					// sent successfully
				case <-ctx.Done():
					log.Println("Replicator context cancelled during send")
					return
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
	// Create UDP listener and setup db to catch files
	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		log.Printf("Listening on %s:%d failed! Got: %v\n", ip, port, err)
		return
	}
	log.Printf("Listening on %s:%d", ip, port)
	defer func(conn *net.UDPConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("Error closing listener: %v\n", err)
		}
	}(conn)
	// Start the loop and check context for done, otherwise listen for packets
	for {
		select {
		case <-ctx.Done():
			log.Println("Proxy Listener exiting due to signal")
			return
		default:
			payload := make([]byte, udpMaxBufferSize)
			timeout := time.Now().Add(5 * time.Second)
			err := conn.SetReadDeadline(timeout)
			if err != nil {
				log.Printf("Issue setting deadline: %v", err)
				return
			}
			length, fromIP, err := conn.ReadFromUDP(payload)
			if err != nil {
				// No packets received before deadline moving on ...
				continue
			}
			payload = payload[:length]
			if verbose {
				log.Printf("Packet Received from %s with size of %d", fromIP.String(), length)
			}
			// Send payload to the proxyChan channel
			select {
			case proxyChan <- payload:
			case <-ctx.Done():
				return
			default:
				if verbose {
				log.Printf("proxyListener: dropped packet (proxyChan full)")
			}
			}
		}
	}
}

// statsPrinter prints out the status every 10 seconds.
func statsPrinter(ctx context.Context, wg *sync.WaitGroup, rStats *models.RecordStat) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			//log.Println("statsPrinter exiting due to signal")
			return
		case <-time.After(time.Second * 10):
			log.Printf("Netflow v9 Packets: %d Ignored Packets: %d ",
				rStats.LoadValid(), rStats.LoadInvalid())
		}
	}
}

// Ingest pulls byte payload off the data chan and puts them in the badger db
func parseNetflow(ctx context.Context, wg *sync.WaitGroup, proxyChan <-chan []byte, dataChan chan<- []byte, rStats *models.RecordStat, verbose bool) {
	defer wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Netflow parser exiting due to signal")
			return
		case payload := <-proxyChan:
			ok, err := netflow.IsValidNetFlow(payload, 9)
		if err != nil {
			log.Printf("Skipping packet due to issue parsing: %v", err)
			rStats.IncrInvalid()
		} else if ok {
			rStats.IncrValid()
			select {
			case dataChan <- payload:
			case <-ctx.Done():
				log.Println("Netflow parser context cancelled during send")
				return
			default:
				if verbose {
					log.Printf("Netflow parser: dropped packet (dataChan full)")
				}
			}
		} else {
			rStats.IncrInvalid()
		}
		case <-ticker.C:
			log.Printf("Netflow v9 Packets: %d Ignored Packets: %d",
				rStats.LoadValid(), rStats.LoadInvalid())
		}
	}
}

// Run Replay. Kicks off the replay of netflow packets from a db.
func Run(ip string, port int, verbose bool, targets []string) {
	mgr := lifecycle.New()
	ctx := mgr.Context()
	wg := mgr.WaitGroup()

	// Create channels
	proxyChan := make(chan []byte, bufferSize)
	dataChan := make(chan []byte, bufferSize)
	rStats := models.RecordStat{
		ValidCount:   0,
		InvalidCount: 0,
	}
	// Create dedicated channel per target <= 10
	workers := len(targets)
	if workers == 0 {
		log.Fatal("Error: at least one --target is required (format: IP:PORT)")
	}
	if workers > 10 {
		log.Println("Can't have more than 10 Targets")
		os.Exit(1)
	}
	workerChans := make([]chan []byte, workers)
	// start workers
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		id := w + 1
		workerChan := make(chan []byte, bufferSize)
		workerChans[w] = workerChan
		target := targets[w]
		targetIP, targetPort, err := net.SplitHostPort(target)
		if err != nil {
			log.Fatalf("Issue parsing target: %v\n", err)
		}
		targetPortInt, err := strconv.Atoi(targetPort)
		if err != nil {
			log.Fatalf("Issue parsing target port: %v\n", err)
		}
		if targetPortInt < 1 || targetPortInt > 65535 {
			log.Fatalf("Error: target port %d out of valid range (1-65535)", targetPortInt)
		}
		go worker(id, ctx, targetIP, targetPortInt, wg, workerChan)
	}

	// Start parseNetflow and replicator first
	wg.Add(1)
	go statsPrinter(ctx, wg, &rStats)
	wg.Add(1)
	go parseNetflow(ctx, wg, proxyChan, dataChan, &rStats, verbose)
	wg.Add(1)
	go replicator(ctx, wg, dataChan, workerChans, verbose)

	// Finally, start up proxyListener
	wg.Add(1)
	go proxyListener(ctx, wg, ip, port, proxyChan, verbose)

	// Setup signal handling via lifecycle manager
	cleanupDone := mgr.SetupSignalHandler()
	<-cleanupDone
	mgr.Wait()
	return
}
