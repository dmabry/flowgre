// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package stats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dmabry/flowgre/models"
)

// newTestCollector creates a Collector ready for testing.
func newTestCollector() *Collector {
	return &Collector{
		StatsChan: make(chan models.WorkerStat, 10),
		StatsMap:  make(map[int]models.WorkerStat),
		StartTime: time.Now(),
		Config: &models.Config{
			Protocol: "netflow",
		},
	}
}

func TestCollector_History_Appends(t *testing.T) {
	t.Parallel()

	sc := newTestCollector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go sc.Run(&wg, ctx)

	// Send a stat to trigger a history snapshot
	sc.StatsChan <- models.WorkerStat{
		WorkerID:  1,
		SourceID:  100,
		FlowsSent: 10,
		Cycles:    1,
		BytesSent: 1024,
	}

	// Give the collector a moment to process
	time.Sleep(100 * time.Millisecond)

	sc.mu.RLock()
	historyLen := len(sc.History)
	sc.mu.RUnlock()

	if historyLen == 0 {
		t.Error("expected at least one history snapshot, got 0")
	}
}

func TestCollector_History_Rotates(t *testing.T) {
	t.Parallel()

	sc := newTestCollector()
	// Use a large buffer so the test doesn't block on sends
	sc.StatsChan = make(chan models.WorkerStat, 500)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go sc.Run(&wg, ctx)

	// Send enough stats to exceed MaxHistory (300)
	for i := 0; i < 350; i++ {
		sc.StatsChan <- models.WorkerStat{
			WorkerID:  1,
			SourceID:  100,
			FlowsSent: uint64(i + 1),
			Cycles:    uint64(i + 1),
			BytesSent: uint64((i + 1) * 1024),
		}
	}

	// Give the collector time to process all stats
	time.Sleep(500 * time.Millisecond)
	cancel()
	wg.Wait()

	sc.mu.RLock()
	historyLen := len(sc.History)
	sc.mu.RUnlock()

	if historyLen > MaxHistory {
		t.Errorf("history length %d exceeds MaxHistory %d", historyLen, MaxHistory)
	}

	if historyLen == 0 {
		t.Error("expected non-empty history after sending stats")
	}
}

func TestHistoryHandler(t *testing.T) {
	t.Parallel()

	sc := newTestCollector()

	// Pre-populate history
	sc.History = []models.StatSnapshot{
		{
			Timestamp: time.Now(),
			Totals: models.StatTotals{
				FlowsSent: 100,
				Cycles:    10,
				BytesSent: 10240,
			},
			Workers: map[int]models.WorkerStat{
				1: {WorkerID: 1, FlowsSent: 100, Cycles: 10, BytesSent: 10240},
			},
		},
	}

	req := httptest.NewRequest("GET", "/stats/history", nil)
	rec := httptest.NewRecorder()

	sc.HistoryHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var history []models.StatSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &history); err != nil {
		t.Fatalf("failed to unmarshal history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(history))
	}

	if history[0].Totals.FlowsSent != 100 {
		t.Errorf("expected flows_sent 100, got %d", history[0].Totals.FlowsSent)
	}
}

func TestStatsHandler_JSON_Format(t *testing.T) {
	t.Parallel()

	sc := newTestCollector()

	// Pre-populate stats
	sc.StatsMap[1] = models.WorkerStat{
		WorkerID:  1,
		SourceID:  100,
		FlowsSent: 50,
		Cycles:    5,
		BytesSent: 5120,
	}
	sc.StatsTotals = models.StatTotals{
		FlowsSent: 50,
		Cycles:    5,
		BytesSent: 5120,
	}

	req := httptest.NewRequest("GET", "/stats", nil)
	rec := httptest.NewRecorder()

	sc.StatsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal stats: %v", err)
	}

	// Verify the response has both "workers" and "totals" keys
	if _, ok := response["workers"]; !ok {
		t.Error("expected 'workers' key in stats response")
	}
	if _, ok := response["totals"]; !ok {
		t.Error("expected 'totals' key in stats response")
	}

	// Verify totals structure
	totals, ok := response["totals"].(map[string]any)
	if !ok {
		t.Fatal("expected 'totals' to be a map")
	}

	if totals["flows_sent"].(float64) != 50 {
		t.Errorf("expected flows_sent 50, got %v", totals["flows_sent"])
	}
}

func TestDashboardHandler_NewFields(t *testing.T) {
	t.Parallel()

	startTime := time.Now().Add(-5 * time.Minute)
	sc := &Collector{
		StatsChan: make(chan models.WorkerStat, 10),
		StatsMap: map[int]models.WorkerStat{
			1: {WorkerID: 1, SourceID: 100, FlowsSent: 1500, Cycles: 100, BytesSent: 15728640},
			2: {WorkerID: 2, SourceID: 200, FlowsSent: 1400, Cycles: 95, BytesSent: 14680064},
		},
		StatsTotals: models.StatTotals{
			FlowsSent: 2900,
			Cycles:    195,
			BytesSent: 30408704,
		},
		StartTime: startTime,
		Config: &models.Config{
			Protocol:         "ipfix",
			Server:           "10.0.0.1",
			DstPort:          9995,
			Workers:          4,
			Delay:            50,
			SrcRange:         "10.0.0.0/8",
			DstRange:         "172.16.0.0/12",
			TemplateInterval: 30,
		},
	}

	req := httptest.NewRequest("GET", "/dashboard", nil)
	rec := httptest.NewRecorder()

	sc.DashboardHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Protocol badge
	if !strings.Contains(body, "ipfix") {
		t.Error("expected 'ipfix' in dashboard HTML")
	}

	// Uptime (5 minutes)
	if !strings.Contains(body, "5m") {
		t.Error("expected '5m' in uptime display")
	}

	// Worker count from config
	if !strings.Contains(body, "4") {
		t.Error("expected worker count '4' in dashboard")
	}

	// Worker stats
	if !strings.Contains(body, "1500") {
		t.Error("expected flows_sent 1500 in worker table")
	}
	if !strings.Contains(body, "1400") {
		t.Error("expected flows_sent 1400 in worker table")
	}

	// Config values
	if !strings.Contains(body, "10.0.0.1") {
		t.Error("expected server IP in config section")
	}
	if !strings.Contains(body, "9995") {
		t.Error("expected port 9995 in config section")
	}

	// Chart.js loaded
	if !strings.Contains(body, "chart.js") {
		t.Error("expected Chart.js CDN script tag")
	}

	// Theme toggle present
	if !strings.Contains(body, "toggleTheme") {
		t.Error("expected theme toggle function")
	}

	// AJAX polling setup
	if !strings.Contains(body, "setInterval") {
		t.Error("expected setInterval for polling")
	}
}

func TestDashboardHandler_EmptyStats(t *testing.T) {
	t.Parallel()

	sc := &Collector{
		StatsChan: make(chan models.WorkerStat, 10),
		StatsMap:  make(map[int]models.WorkerStat),
		StartTime: time.Now(),
		Config: &models.Config{
			Protocol: "netflow",
			Workers:  2,
		},
	}

	req := httptest.NewRequest("GET", "/dashboard", nil)
	rec := httptest.NewRecorder()

	sc.DashboardHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Should render without error even with no worker stats
	if !strings.Contains(body, "netflow") {
		t.Error("expected 'netflow' in dashboard HTML")
	}
}

func TestDashboardHandler_FormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    uint64
		contains string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 500, "500.0 B"},
		{"kilobytes", 15360, "15.0 KB"},
		{"megabytes", 15728640, "15.0 MB"},
		{"gigabytes", 10737418240, "10.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Collector{
				StatsChan: make(chan models.WorkerStat, 10),
				StatsMap:  make(map[int]models.WorkerStat),
				StartTime: time.Now(),
				Config: &models.Config{
					Protocol: "netflow",
					Workers:  1,
				},
				StatsTotals: models.StatTotals{
					BytesSent: tt.bytes,
				},
			}

			req := httptest.NewRequest("GET", "/dashboard", nil)
			rec := httptest.NewRecorder()

			sc.DashboardHandler(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, tt.contains) {
				t.Errorf("expected formatBytes(%d) to contain %q in HTML", tt.bytes, tt.contains)
			}
		})
	}
}

func TestHumanizeDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"hours", 2*time.Hour + 15*time.Minute + 10*time.Second, "2h 15m 10s"},
		{"days", 3*24*time.Hour + 5*time.Hour + 10*time.Minute, "3d 5h 10m 0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanizeDuration(tt.duration)
			if result != tt.want {
				t.Errorf("humanizeDuration(%v) = %q, want %q", tt.duration, result, tt.want)
			}
		})
	}
}
