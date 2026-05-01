// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package utils provides general-purpose utility functions for flowgre.
// Random number generation, IP math, and packet sending have been extracted
// to dedicated sub-packages (rand.go, ip.go, packet.go).
package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/web/templates"
	"html/template"
)

// BinaryDecoder decodes the given payload from a binary stream into multiple destinations.
func BinaryDecoder(payload io.Reader, dests ...interface{}) error {
	for _, dest := range dests {
		err := binary.Read(payload, binary.BigEndian, dest)
		if err != nil {
			return err
		}
	}
	return nil
}

// ToBytes converts an interface to a gob-encoded byte stream.
func ToBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Constants used for calculating byte sizing for output
const (
	sizeKB = uint64(1 << (10 * 1))
	sizeMB = uint64(1 << (10 * 2))
	sizeGB = uint64(1 << (10 * 3))
)

// StatCollector is used to gather stats about barrage and emit those stats via stdout and web ui.
type StatCollector struct {
	StatsMap    map[int]models.WorkerStat
	StatsChan   chan models.WorkerStat
	StatsTotals models.StatTotals
	Config      *models.Config
}

// Run starts the stat collection loop.
func (sc *StatCollector) Run(wg *sync.WaitGroup, ctx context.Context) {
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
				sc.StatsTotals.Cycles += stat.Cycles
				sc.StatsTotals.FlowsSent += stat.FlowsSent
				sc.StatsTotals.BytesSent += stat.BytesSent
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

// StatsHandler emits worker stats as JSON.
func (sc *StatCollector) StatsHandler(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(sc.StatsMap)
	if err != nil {
		log.Fatalf("Web server had an issue: %v\n", err)
	}
}

// DashboardHandler renders the dashboard HTML page.
func (sc *StatCollector) DashboardHandler(w http.ResponseWriter, r *http.Request) {
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
func (sc *StatCollector) Stop() {
	close(sc.StatsChan)
}
