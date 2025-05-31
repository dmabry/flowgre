// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Proxy is used to accept flows and relay them to multiple targets

package proxy

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/dmabry/flowgre/flow/netflow"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/utils"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
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
		log.Fatal("Listen:", err)
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
			err := binary.Write(&buf, binary.BigEndian, &payload)
			if err != nil {
				log.Printf("Worker [%2d] Issue reading data: %v\n", id, err)
			}
			_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			if err != nil {
				log.Fatalf("Worker [%2d] Issue sending packet %v\n", id, err)
			}
		}
	}
}

// replicator is used to take payloads off the dataChan and pass it to each worker's channel for sending
func replicator(ctx context.Context, wg *sync.WaitGroup, dataChan <-chan []byte, targets []chan []byte, verbose bool) {
	defer wg.Done()
	// Start the loop and check context for done, otherwise listen for packets
	for {
		select {
		case <-ctx.Done():
			log.Println("Replicator exiting due to signal")
			return
		// Validated received and needs to be passed on to workers
		case payload := <-dataChan:
			for _, target := range targets {
				target <- payload
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
		log.Fatalf("Listening on %s:%d failed! Got: %v", ip, port, err)
	}
	log.Printf("Listening on %s:%d", ip, port)
	defer func(conn *net.UDPConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("Error closing listener: %v", err)
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
			proxyChan <- payload
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
				rStats.ValidCount, rStats.InvalidCount)
		}
	}
}

// Ingest pulls byte payload off the data chan and puts them in the badger db
func parseNetflow(ctx context.Context, wg *sync.WaitGroup, proxyChan <-chan []byte, dataChan chan<- []byte, rStats *models.RecordStat, verbose bool) {
	defer wg.Done()
	// Start the loop
	for {
		// Check to see if context is done and return, otherwise pull payloads and write
		select {
		case <-ctx.Done():
			log.Println("Netflow parser exiting due to signal")
			return
		case payload := <-proxyChan:
			// Decode first uint16 and see if it is a version 9
			ok, err := netflow.IsValidNetFlow(payload, 9)
			if err != nil {
				log.Printf("Skipping packet due to issue parsing: %v", err)
			}
			if ok {
				// Netflow v9 Packet send it on
				rStats.ValidCount++
				dataChan <- payload
			} else {
				// Not a Netflow v9 Packet... skip
				rStats.InvalidCount++
			}
		case <-time.After(time.Second * 30):
			log.Printf("No flow packets received for 30s...waiting")
		}
	}
}

// Run Replay. Kicks off the replay of netflow packets from a db.
func Run(ip string, port int, verbose bool, targets []string) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	// Create channels
	proxyChan := make(chan []byte, bufferSize)
	dataChan := make(chan []byte, bufferSize)
	rStats := models.RecordStat{
		ValidCount:   0,
		InvalidCount: 0,
	}
	// Create dedicated channel per target <= 10
	workers := len(targets)
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
	go proxyListener(ctx, wg, ip, port, proxyChan, verbose)

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt, os.Kill, os.Signal(syscall.SIGTERM), os.Signal(syscall.SIGHUP))

	go func() {
		for {
			select {
			case <-signalChan:
				log.Printf("\rReceived signal, shutting down...\n\n")
				cancel()
				cleanupDone <- true
			case <-ctx.Done():
				cleanupDone <- true
			}
		}
	}()
	<-cleanupDone
	wg.Wait()
	close(signalChan)
	close(cleanupDone)
	return
}
