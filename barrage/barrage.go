// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Barrage is used to setup a continuous stream of netflow packets towards a single collector

package barrage

import (
	"context"
	"fmt"
	"github.com/dmabry/flowgre/flow/netflow"
	"github.com/dmabry/flowgre/utils"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

type WorkerStats struct {
	SourceID  int
	FlowsSent uint64
	Cycles    uint64
	BytesSent uint64
}

type Config struct {
	Server    string
	DstPort   int
	Workers   int
	Delay     int
	WebIP     string
	WebPort   int
	Web       bool
	WaitGroup sync.WaitGroup
	Context   context.Context
}

const (
	sizeKB = uint64(1 << (10 * 1))
	sizeMB = uint64(1 << (10 * 2))
	sizeGB = uint64(1 << (10 * 3))
)

// Worker is the goroutine used to create workers
func worker(id int, ctx context.Context, server string, port int, sourceID int, delay int, wg *sync.WaitGroup) {
	defer wg.Done()
	wStats := WorkerStats{
		SourceID:  sourceID,
		FlowsSent: 0,
		Cycles:    0,
		BytesSent: 0,
	}

	startTime := time.Now().UnixNano()
	limiter := time.Tick(time.Millisecond * time.Duration(delay))

	// Configure connection to use.  It looks like a listener, but it will be used to send packet.  Allows me to set the source port
	srcPort := utils.RandomNum(10000, 15000)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Fatal("Listen:", err)
	}
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(server)

	// Generate and send Template Flow(s)
	tFlow := netflow.GenerateTemplateNetflow(sourceID)
	tBuf := tFlow.ToBytes()
	log.Printf("Worker [%d] Sending Template Flow\n", id)
	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)
	if err != nil {
		fmt.Errorf("Worker [%d] Issue sending packet %v\n", id, err)
		fmt.Println(err.Error())
	}

	log.Printf("Worker [%d] Slinging packets at %s:%d with Source ID: %d and delay of %dms \n",
		id, server, port, sourceID, delay)
	//Infinite loop to keep slinging until we receive context done.
	printStats := false
	sizeLabel := "bytes"
	var sizeOut uint64 = 0
	for {
		now := time.Now().UnixNano()
		statsCycle := (now - startTime) / int64(time.Second) % 30
		// Print out basic statistics per worker every 30 seconds
		if statsCycle == 0 {
			if printStats {
				switch {
				case wStats.BytesSent >= sizeKB && wStats.BytesSent <= sizeMB:
					sizeLabel = "KB"
					sizeOut = wStats.BytesSent / sizeKB
				case wStats.BytesSent >= sizeMB && wStats.BytesSent <= sizeGB:
					sizeLabel = "MB"
					sizeOut = wStats.BytesSent / sizeMB
				case wStats.BytesSent > sizeGB:
					sizeLabel = "GB"
					sizeOut = wStats.BytesSent / sizeGB
				default:
					sizeOut = wStats.BytesSent
				}
				log.Printf("Worker [%d] Cycles: %d Flows Sent: %d Bytes Sent: %d %s\n", id, wStats.Cycles, wStats.FlowsSent, sizeOut, sizeLabel)
				printStats = false
			}
		} else {
			printStats = true
		}
		select {
		case <-ctx.Done(): //Caught the signal to be done.... time to wrap it up
			log.Printf("Worker [%d] Exiting due to signal\n", id)
			return
		default:
			// Basic limiter to throttle/delay packets
			<-limiter
			flowCount := utils.RandomNum(20, 150)
			flow := netflow.GenerateDataNetflow(flowCount, sourceID)
			buf := flow.ToBytes()
			bytes, err := utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			if err != nil {
				fmt.Errorf("Worker [%d] Issue sending packet %v\n", id, err)
				fmt.Println(err.Error())
			}
			wStats.FlowsSent += uint64(flowCount)
			wStats.Cycles++
			wStats.BytesSent += uint64(bytes)
		}
	}
}

// Run the Barrage
// func Run(server string, port int, delay int, workers int) {
func Run(config *Config) {
	//waitgroup and context used to track and control workers
	//wg := sync.WaitGroup{}
	if &config.WaitGroup == nil {
		config.WaitGroup = sync.WaitGroup{}
	}
	wg := &config.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	config.Context = ctx

	// TODO: I'm thinking about using a chan to return results...
	// results := make(chan string, 1000)

	// Start up the workers
	wg.Add(config.Workers)
	for w := 1; w <= config.Workers; w++ {
		sourceID := utils.RandomNum(100, 10000)
		go worker(w, ctx, config.Server, config.DstPort, sourceID, config.Delay, wg)
	}

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			log.Printf("\nReceived an interrupt, closing connections...\n\n")
			// Cancel workers via context
			cancel()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
	wg.Wait()
	os.Exit(0)
}
