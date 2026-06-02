// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Barrage is used to set up a continuous stream of flow packets towards a single collector.
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

// worker is the generic goroutine used to create workers for any FlowGenerator.
func worker(id int, ctx context.Context, server string, port int, srcRange string, dstRange string, sourceID int, delay int, templateInterval int, wg *sync.WaitGroup, statsChan chan<- models.WorkerStat, gen FlowGenerator) {
	defer wg.Done()
	label := gen.Label()
	wStats := models.WorkerStat{
		WorkerID:  id,
		SourceID:  sourceID,
		FlowsSent: 0,
		Cycles:    0,
		BytesSent: 0,
	}

	// Configure connection to use. It looks like a listener, but it will be used to send packet. Allows setting the source port.
	srcPort := utils.RandomNum(sourcePortMin, sourcePortMax)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Printf("%s [%2d] Listen failed: %v", label, id, err)
		return
	}
	defer conn.Close()

	// Convert given IP String to net.IP type
	destIP := net.ParseIP(server)
	// start new Session for this worker
	session := netflow.NewSession()

	// Generate and send first Template Flow(s)
	tBuf := gen.GenerateTemplate(sourceID, session)
	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)
	if err != nil {
		log.Printf("%s [%2d] Issue sending initial packet: %v", label, id, err)
		return
	}

	// Generate and send Options Data (IPFIX only; returns nil for NetFlow)
	oBuf := gen.GenerateOptionsData(sourceID, session)
	if oBuf != nil {
		_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, oBuf, false)
		if err != nil {
			log.Printf("%s [%2d] Issue sending options data packet: %v", label, id, err)
			return
		}
	}

	log.Printf("%s [%2d] Slinging packets at %s:%d with Source ID: %5d and delay of %dms\n",
		label, id, server, port, sourceID, delay)

	// Data limiter throttles flow packet generation.
	dataLimiter := time.NewTicker(time.Millisecond * time.Duration(delay))
	defer dataLimiter.Stop()

	// Template retransmission ticker — fires every templateInterval seconds.
	// When templateInterval is 0, no ticker is created so templates are never retransmitted.
	var tmplTicker *time.Ticker
	if templateInterval > 0 {
		tmplTicker = time.NewTicker(time.Duration(templateInterval) * time.Second)
		defer tmplTicker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("%s [%2d] Exiting due to signal\n", label, id)
			return
		case <-tmplTicker.C:
			// Retransmit template every templateInterval seconds
			bytes, err := utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)
			if err != nil {
				log.Printf("%s [%2d] Issue sending template packet: %v", label, id, err)
				return
			}
			wStats.FlowsSent++
			wStats.BytesSent += uint64(bytes)
			statsChan <- wStats
		case <-dataLimiter.C:
			flowCount := utils.RandomNum(5, 25)
			buf := gen.GenerateData(flowCount, sourceID, srcRange, dstRange, session)
			bytes, err := utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			if err != nil {
				log.Printf("%s [%2d] Issue sending data packet: %v", label, id, err)
				return
			}
			wStats.FlowsSent += uint64(flowCount)
			wStats.Cycles++
			wStats.BytesSent += uint64(bytes)
		}
	}
}

// RunCtx starts the barrage with the given config, FlowGenerator, and external
// context. Cancelling ctx stops all workers cleanly. Use Run() for CLI usage
// where OS signal handling is desired.
func RunCtx(ctx context.Context, config *models.Config, gen FlowGenerator) {
	wg := &sync.WaitGroup{}

	buffer := 20
	// Start the StatsCollector
	sc := &stats.Collector{
		StartTime: time.Now(),
	}
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

	// Default template retransmission interval is 30 seconds if not set
	templateInterval := config.TemplateInterval
	if templateInterval <= 0 {
		templateInterval = 30
	}

	// Start up the workers
	wg.Add(config.Workers)
	for w := 1; w <= config.Workers; w++ {
		sourceID := utils.RandomNum(sourceIDMin, sourceIDMax)
		go worker(w, ctx, config.Server, config.DstPort, config.SrcRange, config.DstRange, sourceID, config.Delay, templateInterval, wg, sc.StatsChan, gen)
	}

	// Start WebServer if needed
	if config.Web {
		wg.Add(1)
		go web.RunWebServer(config.WebIP, config.WebPort, wg, ctx, sc)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	sc.Stop()
}

// Run starts the barrage with the given config and FlowGenerator.
// It sets up OS signal handling (SIGINT/SIGTERM) for clean shutdown.
// Use RunCtx() when you need to control the lifecycle via context.
func Run(config *models.Config, gen FlowGenerator) {
	mgr := lifecycle.New()
	ctx := mgr.Context()
	wg := mgr.WaitGroup()

	buffer := 20
	// Start the StatsCollector
	sc := &stats.Collector{
		StartTime: time.Now(),
	}
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

	// Default template retransmission interval is 30 seconds if not set
	templateInterval := config.TemplateInterval
	if templateInterval <= 0 {
		templateInterval = 30
	}

	// Start up the workers
	wg.Add(config.Workers)
	for w := 1; w <= config.Workers; w++ {
		sourceID := utils.RandomNum(sourceIDMin, sourceIDMax)
		go worker(w, ctx, config.Server, config.DstPort, config.SrcRange, config.DstRange, sourceID, config.Delay, templateInterval, wg, sc.StatsChan, gen)
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
