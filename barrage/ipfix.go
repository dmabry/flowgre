// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package barrage implements continuous flow generation for NetFlow and IPFIX.
package barrage

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/stats"
	"github.com/dmabry/flowgre/utils"
	"github.com/dmabry/flowgre/web"
)

const (
	ipfixSourcePortMin = 10000
	ipfixSourcePortMax = 15000
	ipfixSourceIDMin   = 100
	ipfixSourceIDMax   = 10000
)

// ipfixWorker sends a continuous stream of IPFIX packets.
func ipfixWorker(id int, ctx context.Context, server string, port int, srcRange string, dstRange string, sourceID int, delay int, wg *sync.WaitGroup, statsChan chan<- models.WorkerStat) {
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

	srcPort := utils.RandomNum(ipfixSourcePortMin, ipfixSourcePortMax)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Printf("IPFIX Worker [%2d] Listen failed: %v", id, err)
		return
	}
	defer conn.Close()

	destIP := net.ParseIP(server)
	session := netflow.NewSession()

	// Generate and send first Template Flow
	tFlow := ipfix.GenerateTemplateIPFIX(sourceID, session)
	tBuf := tFlow.ToBytes()
	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)
	if err != nil {
		log.Printf("IPFIX Worker [%2d] Issue sending initial packet: %v", id, err)
		return
	}

	log.Printf("IPFIX Worker [%2d] Slinging packets at %s:%d with Source ID: %5d and delay of %dms\n",
		id, server, port, sourceID, delay)

	takeAction := false
	for {
		now := time.Now().UnixNano()
		cycle := (now - startTime) / int64(time.Second) % 30

		if cycle == 0 {
			if takeAction {
				takeAction = false
				bytes, err := utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, tBuf, false)
				if err != nil {
					log.Printf("IPFIX Worker [%2d] Issue sending template packet: %v", id, err)
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
		case <-ctx.Done():
			log.Printf("IPFIX Worker [%2d] Exiting due to signal\n", id)
			return
		default:
			<-limiter
			flowCount := utils.RandomNum(5, 25)
			flow := ipfix.GenerateDataIPFIX(flowCount, sourceID, srcRange, dstRange, 0, session)
			buf := flow.ToBytes()
			bytes, err := utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			if err != nil {
				log.Printf("IPFIX Worker [%2d] Issue sending data packet: %v", id, err)
				return
			}
			wStats.FlowsSent += uint64(flowCount)
			wStats.Cycles++
			wStats.BytesSent += uint64(bytes)
		}
	}
}

// RunIPFIX starts the IPFIX barrage with the given config.
func RunIPFIX(config *models.Config) {
	mgr := lifecycle.New()
	ctx := mgr.Context()
	wg := mgr.WaitGroup()

	buffer := 20
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

	wg.Add(config.Workers)
	for w := 1; w <= config.Workers; w++ {
		sourceID := utils.RandomNum(ipfixSourceIDMin, ipfixSourceIDMax)
		go ipfixWorker(w, ctx, config.Server, config.DstPort, config.SrcRange, config.DstRange, sourceID, config.Delay, wg, sc.StatsChan)
	}

	if config.Web {
		wg.Add(1)
		go web.RunWebServer(config.WebIP, config.WebPort, wg, ctx, sc)
	}

	cleanupDone := mgr.SetupSignalHandler()
	<-cleanupDone
	mgr.Wait()
	sc.Stop()
}
