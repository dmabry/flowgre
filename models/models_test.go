// Package models provides data structures for flowgre.
// This package contains pure data structures without concurrency primitives,
// except for RecordStat which uses atomic operations for thread-safe counters.
package models

import (
	"sync"
	"testing"
)

// TestConfig verifies Config struct initialization and field access.
func TestConfig(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Server:   "127.0.0.1",
		DstPort:  9995,
		SrcRange: "10.0.0.0/8",
		DstRange: "10.0.0.0/8",
		Workers:  4,
		Delay:    100,
		WebIP:    "0.0.0.0",
		WebPort:  8080,
		Web:      true,
	}

	if cfg.Server != "127.0.0.1" {
		t.Errorf("Config.Server = %q, want %q", cfg.Server, "127.0.0.1")
	}
	if cfg.DstPort != 9995 {
		t.Errorf("Config.DstPort = %d, want %d", cfg.DstPort, 9995)
	}
	if cfg.Workers != 4 {
		t.Errorf("Config.Workers = %d, want %d", cfg.Workers, 4)
	}
	if !cfg.Web {
		t.Error("Config.Web = false, want true")
	}
}

// TestConfigZeroValues verifies that Config handles zero values correctly.
func TestConfigZeroValues(t *testing.T) {
	t.Parallel()
	cfg := &Config{}

	if cfg.Server != "" {
		t.Errorf("Zero Config.Server = %q, want empty", cfg.Server)
	}
	if cfg.DstPort != 0 {
		t.Errorf("Zero Config.DstPort = %d, want 0", cfg.DstPort)
	}
	if cfg.Workers != 0 {
		t.Errorf("Zero Config.Workers = %d, want 0", cfg.Workers)
	}
}

// TestWorkerStat verifies WorkerStat struct initialization and field access.
func TestWorkerStat(t *testing.T) {
	t.Parallel()
	stat := WorkerStat{
		WorkerID:  1,
		SourceID:  100,
		FlowsSent: 1000,
		Cycles:    50,
		BytesSent: 50000,
	}

	if stat.WorkerID != 1 {
		t.Errorf("WorkerStat.WorkerID = %d, want %d", stat.WorkerID, 1)
	}
	if stat.SourceID != 100 {
		t.Errorf("WorkerStat.SourceID = %d, want %d", stat.SourceID, 100)
	}
	if stat.FlowsSent != 1000 {
		t.Errorf("WorkerStat.FlowsSent = %d, want %d", stat.FlowsSent, 1000)
	}
	if stat.Cycles != 50 {
		t.Errorf("WorkerStat.Cycles = %d, want %d", stat.Cycles, 50)
	}
	if stat.BytesSent != 50000 {
		t.Errorf("WorkerStat.BytesSent = %d, want %d", stat.BytesSent, 50000)
	}
}

// TestWorkerStatZeroValues verifies WorkerStat with zero values.
func TestWorkerStatZeroValues(t *testing.T) {
	t.Parallel()
	stat := WorkerStat{}

	if stat.WorkerID != 0 {
		t.Errorf("Zero WorkerStat.WorkerID = %d, want 0", stat.WorkerID)
	}
	if stat.FlowsSent != 0 {
		t.Errorf("Zero WorkerStat.FlowsSent = %d, want 0", stat.FlowsSent)
	}
}

// TestRecordStatIncrValid tests atomic increment of ValidCount.
func TestRecordStatIncrValid(t *testing.T) {
	t.Parallel()
	stat := &RecordStat{}

	// First increment
	val1 := stat.IncrValid()
	if val1 != 1 {
		t.Errorf("First IncrValid() = %d, want 1", val1)
	}
	if stat.LoadValid() != 1 {
		t.Errorf("LoadValid() after first increment = %d, want 1", stat.LoadValid())
	}

	// Second increment
	val2 := stat.IncrValid()
	if val2 != 2 {
		t.Errorf("Second IncrValid() = %d, want 2", val2)
	}
	if stat.LoadValid() != 2 {
		t.Errorf("LoadValid() after second increment = %d, want 2", stat.LoadValid())
	}
}

// TestRecordStatIncrInvalid tests atomic increment of InvalidCount.
func TestRecordStatIncrInvalid(t *testing.T) {
	t.Parallel()
	stat := &RecordStat{}

	val1 := stat.IncrInvalid()
	if val1 != 1 {
		t.Errorf("First IncrInvalid() = %d, want 1", val1)
	}
	if stat.LoadInvalid() != 1 {
		t.Errorf("LoadInvalid() after first increment = %d, want 1", stat.LoadInvalid())
	}

	val2 := stat.IncrInvalid()
	if val2 != 2 {
		t.Errorf("Second IncrInvalid() = %d, want 2", val2)
	}
	if stat.LoadInvalid() != 2 {
		t.Errorf("LoadInvalid() after second increment = %d, want 2", stat.LoadInvalid())
	}
}

// TestRecordStatConcurrentAccess tests thread safety of RecordStat.
func TestRecordStatConcurrentAccess(t *testing.T) {
	t.Parallel()
	stat := &RecordStat{}
	var wg sync.WaitGroup

	numGoroutines := 10
	incrementsPerGoroutine := 100

	// Start multiple goroutines incrementing ValidCount
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				stat.IncrValid()
			}
		}()
	}

	wg.Wait()

	expected := uint64(numGoroutines * incrementsPerGoroutine)
	actual := stat.LoadValid()
	if actual != expected {
		t.Errorf("Concurrent IncrValid() resulted in %d, want %d", actual, expected)
	}
}

// TestRecordStatConcurrentMixedAccess tests thread safety with mixed operations.
func TestRecordStatConcurrentMixedAccess(t *testing.T) {
	t.Parallel()
	stat := &RecordStat{}
	var wg sync.WaitGroup

	numGoroutines := 10

	// Goroutines incrementing ValidCount
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				stat.IncrValid()
			}
		}()
	}

	// Goroutines incrementing InvalidCount
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				stat.IncrInvalid()
			}
		}()
	}

	// Goroutines reading counts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = stat.LoadValid()
				_ = stat.LoadInvalid()
			}
		}()
	}

	wg.Wait()

	expectedValid := uint64(numGoroutines * 50)
	expectedInvalid := uint64(numGoroutines * 30)

	if stat.LoadValid() != expectedValid {
		t.Errorf("LoadValid() = %d, want %d", stat.LoadValid(), expectedValid)
	}
	if stat.LoadInvalid() != expectedInvalid {
		t.Errorf("LoadInvalid() = %d, want %d", stat.LoadInvalid(), expectedInvalid)
	}
}

// TestRecordStatLoadOperations tests LoadValid and LoadInvalid.
func TestRecordStatLoadOperations(t *testing.T) {
	t.Parallel()
	stat := &RecordStat{}

	// Initial values should be 0
	if stat.LoadValid() != 0 {
		t.Errorf("Initial LoadValid() = %d, want 0", stat.LoadValid())
	}
	if stat.LoadInvalid() != 0 {
		t.Errorf("Initial LoadInvalid() = %d, want 0", stat.LoadInvalid())
	}

	// Increment and load
	stat.IncrValid()
	stat.IncrInvalid()

	if stat.LoadValid() != 1 {
		t.Errorf("LoadValid() after increment = %d, want 1", stat.LoadValid())
	}
	if stat.LoadInvalid() != 1 {
		t.Errorf("LoadInvalid() after increment = %d, want 1", stat.LoadInvalid())
	}
}

// TestStatTotals verifies StatTotals struct initialization.
func TestStatTotals(t *testing.T) {
	t.Parallel()
	totals := StatTotals{
		FlowsSent: 10000,
		Cycles:    500,
		BytesSent: 1000000,
	}

	if totals.FlowsSent != 10000 {
		t.Errorf("StatTotals.FlowsSent = %d, want %d", totals.FlowsSent, 10000)
	}
	if totals.Cycles != 500 {
		t.Errorf("StatTotals.Cycles = %d, want %d", totals.Cycles, 500)
	}
	if totals.BytesSent != 1000000 {
		t.Errorf("StatTotals.BytesSent = %d, want %d", totals.BytesSent, 1000000)
	}
}

// TestStatTotalsZeroValues verifies StatTotals with zero values.
func TestStatTotalsZeroValues(t *testing.T) {
	t.Parallel()
	totals := StatTotals{}

	if totals.FlowsSent != 0 {
		t.Errorf("Zero StatTotals.FlowsSent = %d, want 0", totals.FlowsSent)
	}
	if totals.Cycles != 0 {
		t.Errorf("Zero StatTotals.Cycles = %d, want 0", totals.Cycles)
	}
	if totals.BytesSent != 0 {
		t.Errorf("Zero StatTotals.BytesSent = %d, want 0", totals.BytesSent)
	}
}

// TestWorkerStats verifies WorkerStats type (slice of WorkerStat).
func TestWorkerStats(t *testing.T) {
	t.Parallel()
	stats := WorkerStats{
		{WorkerID: 1, FlowsSent: 100},
		{WorkerID: 2, FlowsSent: 200},
		{WorkerID: 3, FlowsSent: 300},
	}

	if len(stats) != 3 {
		t.Errorf("WorkerStats length = %d, want %d", len(stats), 3)
	}

	if stats[0].WorkerID != 1 {
		t.Errorf("stats[0].WorkerID = %d, want %d", stats[0].WorkerID, 1)
	}
	if stats[2].FlowsSent != 300 {
		t.Errorf("stats[2].FlowsSent = %d, want %d", stats[2].FlowsSent, 300)
	}
}

// TestHealth verifies Health struct initialization.
func TestHealth(t *testing.T) {
	t.Parallel()
	health := Health{
		Status:  "OK",
		Message: "Service is running",
	}

	if health.Status != "OK" {
		t.Errorf("Health.Status = %q, want %q", health.Status, "OK")
	}
	if health.Message != "Service is running" {
		t.Errorf("Health.Message = %q, want %q", health.Message, "Service is running")
	}
}

// TestHealthZeroValues verifies Health with zero values.
func TestHealthZeroValues(t *testing.T) {
	t.Parallel()
	health := Health{}

	if health.Status != "" {
		t.Errorf("Zero Health.Status = %q, want empty", health.Status)
	}
	if health.Message != "" {
		t.Errorf("Zero Health.Message = %q, want empty", health.Message)
	}
}

// TestDashboardPage verifies DashboardPage struct initialization.
func TestDashboardPage(t *testing.T) {
	t.Parallel()
	cfg := &Config{Server: "127.0.0.1", Workers: 4}
	statsMap := map[int]WorkerStat{1: {WorkerID: 1, FlowsSent: 100}}
	totals := StatTotals{FlowsSent: 100}

	page := DashboardPage{
		Title:       "Flowgre Dashboard",
		Comment:     "Test dashboard",
		HealthOut:   Health{Status: "OK"},
		ConfigOut:   cfg,
		StatsMapOut: statsMap,
		StatsTotal:  totals,
	}

	if page.Title != "Flowgre Dashboard" {
		t.Errorf("DashboardPage.Title = %q, want %q", page.Title, "Flowgre Dashboard")
	}
	if page.ConfigOut != cfg {
		t.Error("DashboardPage.ConfigOut does not point to expected Config")
	}
	if len(page.StatsMapOut) != 1 {
		t.Errorf("DashboardPage.StatsMapOut length = %d, want %d", len(page.StatsMapOut), 1)
	}
	if page.StatsTotal.FlowsSent != 100 {
		t.Errorf("DashboardPage.StatsTotal.FlowsSent = %d, want %d", page.StatsTotal.FlowsSent, 100)
	}
}

// TestDashboardPageZeroValues verifies DashboardPage with zero values.
func TestDashboardPageZeroValues(t *testing.T) {
	t.Parallel()
	page := DashboardPage{}

	if page.Title != "" {
		t.Errorf("Zero DashboardPage.Title = %q, want empty", page.Title)
	}
	if page.ConfigOut != nil {
		t.Errorf("Zero DashboardPage.ConfigOut = %v, want nil", page.ConfigOut)
	}
	if page.StatsMapOut == nil {
		// StatsMapOut being nil is acceptable for zero value
	}
}
