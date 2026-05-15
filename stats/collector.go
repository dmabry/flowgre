// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package stats provides worker statistics collection for flowgre barrage mode.
package stats

import (
	"context"
	"encoding/json"
	"fmt"
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

	// MaxHistory caps the rolling history buffer (300 snapshots at 2s intervals = 10 minutes).
	MaxHistory = 300
)

// Collector gathers stats about barrage workers and emits them via stdout and web UI.
type Collector struct {
	mu          sync.RWMutex
	StatsMap    map[int]models.WorkerStat
	StatsChan   chan models.WorkerStat
	StatsTotals models.StatTotals
	Config      *models.Config
	StartTime   time.Time             // when the barrage started
	History     []models.StatSnapshot // rolling history of stat snapshots
}

// Run starts the stat collection loop. It reads from StatsChan and aggregates totals.
// Uses a ticker-driven pattern instead of a busy-wait loop to avoid wasting CPU cycles.
func (sc *Collector) Run(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	log.Println("Stats Collector started")
	sizeLabel := "bytes"
	var sizeOut uint64

	// Periodic ticker for periodic logging/heartbeat (unused here but keeps
	// the loop from blocking indefinitely when no stats arrive).
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

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
				sc.mu.Lock()
				sc.StatsMap[stat.WorkerID] = stat
				// Recalculate totals from map to avoid double-counting cumulative stats
				sc.StatsTotals = models.StatTotals{}
				for _, s := range sc.StatsMap {
					sc.StatsTotals.Cycles += s.Cycles
					sc.StatsTotals.FlowsSent += s.FlowsSent
					sc.StatsTotals.BytesSent += s.BytesSent
				}
				// Append a history snapshot
				sc.appendSnapshot()
				sc.mu.Unlock()
			} else {
				log.Println("Stats Channel Closed!")
				return
			}
		case <-ctx.Done():
			log.Printf("Stats Collector Exiting due to signal\n")
			return
		case <-ticker.C:
			// Periodic tick — nothing to do, just keeps the loop alive
			// so it doesn't block forever when no stats arrive.
		}
	}
}

// appendSnapshot appends a point-in-time snapshot to the rolling history buffer.
// Must be called with sc.mu held (write lock).
func (sc *Collector) appendSnapshot() {
	workersCopy := make(map[int]models.WorkerStat, len(sc.StatsMap))
	for k, v := range sc.StatsMap {
		workersCopy[k] = v
	}
	snapshot := models.StatSnapshot{
		Timestamp: time.Now(),
		Totals:    sc.StatsTotals,
		Workers:   workersCopy,
	}
	sc.History = append(sc.History, snapshot)
	if len(sc.History) > MaxHistory {
		sc.History = sc.History[len(sc.History)-MaxHistory:]
	}
}

// StatsHandler emits worker stats as JSON for the web API.
func (sc *Collector) StatsHandler(w http.ResponseWriter, r *http.Request) {
	sc.mu.RLock()
	statsCopy := make(map[int]models.WorkerStat, len(sc.StatsMap))
	for k, v := range sc.StatsMap {
		statsCopy[k] = v
	}
	totalsCopy := sc.StatsTotals
	sc.mu.RUnlock()

	// Return both per-worker stats and totals in a single response for the dashboard
	response := map[string]any{
		"workers": statsCopy,
		"totals":  totalsCopy,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Web server had an issue: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HistoryHandler returns time-series stats for charting.
func (sc *Collector) HistoryHandler(w http.ResponseWriter, r *http.Request) {
	sc.mu.RLock()
	historyCopy := make([]models.StatSnapshot, len(sc.History))
	copy(historyCopy, sc.History)
	sc.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(historyCopy)
	if err != nil {
		log.Printf("Web server had an issue: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// DashboardHandler renders the dashboard HTML page with current stats.
func (sc *Collector) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	sc.mu.RLock()
	statsCopy := make(map[int]models.WorkerStat, len(sc.StatsMap))
	for k, v := range sc.StatsMap {
		statsCopy[k] = v
	}
	totalsCopy := sc.StatsTotals
	sc.mu.RUnlock()

	// Calculate uptime
	var uptimeStr string
	if !sc.StartTime.IsZero() {
		uptimeStr = humanizeDuration(time.Since(sc.StartTime))
	}

	protocol := ""
	if sc.Config != nil {
		protocol = sc.Config.Protocol
	}

	d := models.DashboardPage{
		Title:   "Flowgre Dashboard",
		Comment: "Basic metrics about flowgre",
		HealthOut: models.Health{
			Status:  "OK",
			Message: "Flowgre is Flinging Packets!",
		},
		ConfigOut:   sc.Config,
		StatsMapOut: statsCopy,
		StatsTotal:  totalsCopy,
		Protocol:    protocol,
		StartTime:   sc.StartTime,
		Uptime:      uptimeStr,
	}

	t, err := template.New("dashboard").Funcs(template.FuncMap{
		"formatBytes": func(bytes uint64) string {
			if bytes == 0 {
				return "0 B"
			}
			const unit = 1024
			const units = "BKMG"
			i := 0
			f := float64(bytes)
			for f >= unit && i < len(units)-1 {
				f /= unit
				i++
			}
			return fmt.Sprintf("%.1f %sB", f, string(units[i]))
		},
	}).Parse(templates.DashboardTpl)
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

// humanizeDuration formats a duration as a human-readable string.
func humanizeDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
