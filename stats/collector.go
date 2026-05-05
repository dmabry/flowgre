// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package stats provides worker statistics collection for flowgre barrage mode.
package stats

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/web/templates"
	"html/template"
)

// Constants used for calculating byte sizing for output.
const (
	sizeKB = uint64(1 << (10 * 1))
	sizeMB = uint64(1 << (10 * 2))
	sizeGB = uint64(1 << (10 * 3))
)

// Collector gathers stats about barrage workers and emits them via stdout and web UI.
type Collector struct {
	StatsMap    map[int]models.WorkerStat
	StatsChan   chan models.WorkerStat
	StatsTotals models.StatTotals
	Config      *models.Config
}

// Run starts the stat collection loop. It reads from StatsChan and aggregates totals.
func (sc *Collector) Run(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	limiter := time.Tick(time.Second * 5)
	log.Println("Stats Collector started")
	sizeLabel := "bytes"
	var sizeOut uint64
	for {
		select {
		case stat, ok := <-sc.StatsChan:
			if ok {
				switch {
				case stat.BytesSent >= sizeKB && stat.BytesSent <= sizeMB:
					sizeLabel = "KB"
					sizeOut = stat.BytesSent / sizeKB
				case stat.BytesSent >= sizeMB && stat.BytesSent <= sizeGB:
					sizeLabel = "MB"
					sizeOut = stat.BytesSent / sizeMB
				case stat.BytesSent > sizeGB:
					sizeLabel = "GB"
					sizeOut = stat.BytesSent / sizeGB
				default:
					sizeOut = stat.BytesSent
				}
				log.Printf("Worker [%2d] SourceID: %4d Cycles: %d Flows Sent: %d Bytes Sent: %d %s\n",
					stat.WorkerID, stat.SourceID, stat.Cycles, stat.FlowsSent, sizeOut, sizeLabel)
				sc.StatsMap[stat.WorkerID] = stat
				// Recalculate totals from map to avoid double-counting cumulative stats
				sc.StatsTotals = models.StatTotals{}
				for _, s := range sc.StatsMap {
					sc.StatsTotals.Cycles += s.Cycles
					sc.StatsTotals.FlowsSent += s.FlowsSent
					sc.StatsTotals.BytesSent += s.BytesSent
				}
			} else {
				log.Println("Stats Channel Closed!")
			}
		case <-ctx.Done():
			log.Printf("Stats Collector Exiting due to signal\n")
			return
		default:
			<-limiter
		}
	}
}

// StatsHandler emits worker stats as JSON for the web API.
func (sc *Collector) StatsHandler(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(sc.StatsMap)
	if err != nil {
		log.Fatalf("Web server had an issue: %v\n", err)
	}
}

// DashboardHandler renders the dashboard HTML page with current stats.
func (sc *Collector) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	d := models.DashboardPage{
		Title:   "Flowgre Dashboard",
		Comment: "Basic metrics about flowgre",
		HealthOut: models.Health{
			Status:  "OK",
			Message: "Flowgre is Flinging Packets!",
		},
		ConfigOut:   sc.Config,
		StatsMapOut: sc.StatsMap,
		StatsTotal:  sc.StatsTotals,
	}

	t, err := template.New("dashboard").Parse(templates.DashboardTpl)
	if err != nil {
		log.Printf("Web server had issue: %v\n", err)
	} else {
		err = t.Execute(w, d)
		if err != nil {
			log.Printf("Web server had issue: %v\n", err)
		}
	}
}

// Stop closes the stats channel gracefully.
func (sc *Collector) Stop() {
	close(sc.StatsChan)
}
