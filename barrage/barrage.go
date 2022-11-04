// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Barrage is used to set up a continuous stream of netflow packets towards a single collector

package barrage

import (
	"context"
	"github.com/dmabry/flowgre/flow/netflow"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/utils"
	"github.com/dmabry/flowgre/web"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Worker is the goroutine used to create workers
func worker(id int, ctx context.Context, server string, port int, srcRange string, dstRange string, sourceID int, delay int, wg *sync.WaitGroup, statsChan chan<- models.WorkerStat) {
	defer wg.Done()
	wStats := models.WorkerStat{
		WorkerID:  id,
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
	// start new FlowTracker
	ft := new(netflow.FlowTracker).Init()

	// Generate and send first Template Flow(s)
	tFlow := netflow.GenerateTemplateNetflow(sourceID, &ft)
	tBuf := tFlow.ToBytes()
	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)
	if err != nil {
		log.Fatalf("Worker [%2d] Issue sending packet %v\n", id, err)
	}

	log.Printf("Worker [%2d] Slinging packets at %s:%d with Source ID: %5d and delay of %dms \n",
		id, server, port, sourceID, delay)
	//Infinite loop to keep slinging until we receive context done.
	takeAction := false

	for {
		now := time.Now().UnixNano()
		cycle := (now - startTime) / int64(time.Second) % 30
		// Print out basic statistics per worker every 30 seconds
		if cycle == 0 {
			if takeAction {
				takeAction = false
				// Send Template per worker every 30 seconds
				bytes, err := utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)
				if err != nil {
					log.Fatalf("Worker [%2d] Issue sending packet %v\n", id, err)
				}
				wStats.FlowsSent++
				wStats.BytesSent += uint64(bytes)
				statsChan <- wStats
			}
		} else {
			takeAction = true
		}
		select {
		case <-ctx.Done(): //Caught the signal to be done.... time to wrap it up
			log.Printf("Worker [%2d] Exiting due to signal\n", id)
			return
		default:
			// Basic limiter to throttle/delay packets
			<-limiter
			//flowCount := utils.RandomNum(5, 10) // TODO: For some reason this causes random flow seq issues.
			// using hardcoded number for now
			flowCount := 100
			flow := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, &ft)
			buf := flow.ToBytes()
			bytes, err := utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			if err != nil {
				log.Fatalf("Worker [%2d] Issue sending packet %v\n", id, err)
			}
			wStats.FlowsSent += uint64(flowCount)
			wStats.Cycles++
			wStats.BytesSent += uint64(bytes)
		}
	}
}

// Run the Barrage
// func Run(server string, port int, delay int, workers int) {
func Run(config *models.Config) {
	//wait group and context used to track and control workers
	if &config.WaitGroup == nil {
		config.WaitGroup = sync.WaitGroup{}
	}
	wg := &config.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	config.Context = ctx

	buffer := 20
	// Start the StatsCollector
	sc := &utils.StatCollector{}
	sc.StatsChan = make(chan models.WorkerStat, config.Workers+buffer)
	sc.StatsMap = make(map[int]models.WorkerStat)
	sc.StatsTotals = models.StatTotals{
		FlowsSent: 0,
		Cycles:    0,
		BytesSent: 0,
	}
	sc.Config = config
	wg.Add(1)
	go sc.Run(wg, ctx)

	// Start up the workers
	wg.Add(config.Workers)
	for w := 1; w <= config.Workers; w++ {
		sourceID := utils.RandomNum(100, 10000)
		go worker(w, ctx, config.Server, config.DstPort, config.SrcRange, config.DstRange, sourceID, config.Delay, wg, sc.StatsChan)
	}

	// Start WebServer if needed
	if config.Web {
		wg.Add(1)
		go web.RunWebServer(config.WebIP, config.WebPort, wg, ctx, sc)
	}

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			log.Printf("\rReceived an interrupt, closing connections...\n\n")
			// Cancel workers via context
			cancel()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
	wg.Wait()
	sc.Stop()
	os.Exit(0)
}
