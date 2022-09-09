// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Barrage is used to setup a continous stream of netflow packets towards a single collector

package barrage

import (
	"context"
	"fmt"
	"github.com/dmabry/flowgre/flow/netflow"
	"github.com/dmabry/flowgre/utils"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

type WorkerStats struct {
	FlowsSent int
	Cycles    int
}

// Worker is the goroutine used to create workers
func worker(id int, ctx context.Context, server string, port int, sourceID int, delay int, wg *sync.WaitGroup) {
	defer wg.Done()
	wStats := new(WorkerStats)
	wStats.FlowsSent = 0
	wStats.Cycles = 0
	startTime := time.Now().UnixNano()
	limiter := time.Tick(time.Millisecond * time.Duration(delay))

	// Configure connection to use.  It looks like a listener, but it will be used to send packet.  Allows me to set the source port
	rand.Seed(time.Now().UnixNano())
	srcPort := rand.Intn(15000-10000) + 10000

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Fatal("Listen:", err)
		os.Exit(55)
	}
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(server)

	// Generate and send Template Flow(s)
	tFlow := netflow.GenerateTemplateNetflow(sourceID)
	tBuf := tFlow.ToBytes()
	fmt.Printf("Worker [%d] Sending Template Flow\n", id)
	utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)

	log.Printf("Worker [%d] Slinging packets at %s:%d with Source ID: %d and delay of %dms \n",
		id, server, port, sourceID, delay)
	//Infinite loop to keep slinging until we receive context done.
	printStats := false
	for {
		now := time.Now().UnixNano()
		statsCycle := (now - startTime) / int64(time.Second) % 30
		// Print out basic statistics per worker every 30 seconds
		if statsCycle == 0 {
			if printStats {
				log.Printf("Worker [%d] Cycles: %d Flows Sent: %d\n", id, wStats.Cycles, wStats.FlowsSent)
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
			utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			wStats.FlowsSent += flowCount
			wStats.Cycles++
			//log.Printf("Worker [%d] Doing Stuff!\n", id)
			//time.Sleep(time.Duration(utils.RandomNum(1, 5)) * time.Second)
		}
	}
}

// Run the Barrage
func Run(server string, port int, delay int, workers int) {
	//waitgroup and context used to track and control workers
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	// TODO: I'm thinking about using a chan to return results...
	// results := make(chan string, 1000)

	// Start up the workers
	wg.Add(workers)
	for w := 1; w <= workers; w++ {
		sourceID := utils.RandomNum(100, 10000)
		go worker(w, ctx, server, port, sourceID, delay, &wg)
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
