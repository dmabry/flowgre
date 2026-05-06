// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Barrage is used to set up a continuous stream of netflow packets towards a single collector.
package barrage

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/stats"
	"github.com/dmabry/flowgre/utils"
	"github.com/dmabry/flowgre/web"
)

const (
	// sourcePortMin/Max define the range for random source ports
	sourcePortMin = 10000
	sourcePortMax = 15000
	// sourceIDMin/Max define the range for random source IDs
	sourceIDMin = 100
	sourceIDMax = 10000
)

// worker is the goroutine used to create workers.
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

	// Configure connection to use. It looks like a listener, but it will be used to send packet. Allows setting the source port.
	srcPort := utils.RandomNum(sourcePortMin, sourcePortMax)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Printf("Worker [%2d] Listen failed: %v", id, err)
		return
	}
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(server)
	// start new Session for this worker
	session := netflow.NewSession()

	// Generate and send first Template Flow(s)
	tFlow := netflow.GenerateTemplateNetflow(sourceID, session)
	tBuf := tFlow.ToBytes()
	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)
	if err != nil {
		log.Printf("Worker [%2d] Issue sending initial packet: %v", id, err)
		return
	}

	log.Printf("Worker [%2d] Slinging packets at %s:%d with Source ID: %5d and delay of %dms\n",
		id, server, port, sourceID, delay)
	// Infinite loop to keep slinging until we receive context done.
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
					log.Printf("Worker [%2d] Issue sending template packet: %v", id, err)
					return
				}
				wStats.FlowsSent++
				wStats.BytesSent += uint64(bytes)
				statsChan <- wStats
			}
		} else {
			takeAction = true
		}
		select {
		case <-ctx.Done(): // Caught the signal to be done.... time to wrap it up
			log.Printf("Worker [%2d] Exiting due to signal\n", id)
			return
		default:
			// Basic limiter to throttle/delay packets
			<-limiter
			flowCount := utils.RandomNum(5, 25)
			flow := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session)
			buf := flow.ToBytes()
			bytes, err := utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			if err != nil {
				log.Printf("Worker [%2d] Issue sending data packet: %v", id, err)
				return
			}
			wStats.FlowsSent += uint64(flowCount)
			wStats.Cycles++
			wStats.BytesSent += uint64(bytes)
		}
	}
}

// Run the Barrage.
func Run(config *models.Config) {
	mgr := lifecycle.New()
	ctx := mgr.Context()
	wg := mgr.WaitGroup()

	buffer := 20
	// Start the StatsCollector
	sc := &stats.Collector{}
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
		sourceID := utils.RandomNum(sourceIDMin, sourceIDMax)
		go worker(w, ctx, config.Server, config.DstPort, config.SrcRange, config.DstRange, sourceID, config.Delay, wg, sc.StatsChan)
	}

	// Start WebServer if needed
	if config.Web {
		wg.Add(1)
		go web.RunWebServer(config.WebIP, config.WebPort, wg, ctx, sc)
	}

	// Setup signal handling via lifecycle manager
	cleanupDone := mgr.SetupSignalHandler()
	<-cleanupDone
	mgr.Wait()
	sc.Stop()
}
