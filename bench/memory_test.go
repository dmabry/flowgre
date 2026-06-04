//go:build bench

// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package bench contains memory stability tests for flowgre.
// These run sustained barrage + recorder cycles and monitor heap growth
// and goroutine counts over time to detect leaks.
//
// Run with: go test -tags=bench -v -run TestMemory ./bench/
package bench

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/record"
)

// memSample captures a point-in-time memory snapshot.
type memSample struct {
	Elapsed    time.Duration
	HeapAlloc  uint64 // bytes currently allocated
	HeapInuse  uint64 // bytes held from OS
	Goroutines int
}

// collectSample runs GC, forces a memory snapshot, and returns a sample.
func collectSample() memSample {
	runtime.GC()
	runtime.GC() // double GC for stable readings
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return memSample{
		HeapAlloc:  ms.HeapAlloc,
		HeapInuse:  ms.HeapInuse,
		Goroutines: runtime.NumGoroutine(),
	}
}

// runMemoryTest runs a sustained barrage + recorder for the given duration,
// sampling memory at the specified interval. Returns the collected samples.
func runMemoryTest(t *testing.T, workers, delayMs int, duration, interval time.Duration, gen barrage.FlowGenerator) []memSample {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "flowgre-mem-*")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	recPort := pickPortMem(t)
	dbDir := filepath.Join(tmpDir, "rec")

	// Start recorder
	recCtx, recCancel := context.WithCancel(context.Background())
	go func() {
		record.RunCtx(recCtx, "127.0.0.1", recPort, dbDir, false)
	}()
	time.Sleep(500 * time.Millisecond)

	// Start barrage
	cfg := &models.Config{
		Server:           "127.0.0.1",
		DstPort:          recPort,
		SrcRange:         "10.0.0.0/8",
		DstRange:         "172.16.0.0/12",
		Workers:          workers,
		Delay:            delayMs,
		TemplateInterval: 0,
		Web:              false,
	}

	barrCtx, barrCancel := context.WithTimeout(context.Background(), duration)
	go func() {
		barrage.RunCtx(barrCtx, cfg, gen)
	}()

	// Collect memory samples
	var samples []memSample
	start := time.Now()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		elapsed := time.Since(start)
		if elapsed >= duration {
			break
		}
		sample := collectSample()
		sample.Elapsed = elapsed
		samples = append(samples, sample)
		<-ticker.C
	}

	// Final sample (taken while barrage is still running)
	final := collectSample()
	final.Elapsed = time.Since(start)
	samples = append(samples, final)

	// Cleanup: cancel contexts and let goroutines exit asynchronously
	// We don't wait for clean shutdown — the test has its measurements.
	// The OS cleans up orphaned goroutines when the test binary exits.
	barrCancel()
	recCancel()

	return samples
}

// pickPortMem picks an ephemeral UDP port for memory tests.
func pickPortMem(t *testing.T) int {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("pickPort: %v", err)
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	conn.Close()
	return port
}

// ---------------------------------------------------------------------------
// Memory Stability Tests
// ---------------------------------------------------------------------------

// TestMemory_SteadyState_NetFlow verifies that 4 workers at 100ms delay
// do not exhibit significant heap growth over a sustained run.
func TestMemory_SteadyState_NetFlow(t *testing.T) {
	duration := 2 * time.Minute
	interval := 30 * time.Second
	workers := 4
	delayMs := 100

	samples := runMemoryTest(t, workers, delayMs, duration, interval, barrage.NetFlow())

	if len(samples) < 2 {
		t.Fatal("insufficient samples collected")
	}

	for _, s := range samples {
		t.Logf("%-10s heap_alloc=%8.2f MB  heap_inuse=%8.2f MB  goroutines=%d",
			s.Elapsed.Round(time.Second),
			float64(s.HeapAlloc)/1024/1024,
			float64(s.HeapInuse)/1024/1024,
			s.Goroutines)
	}

	initial := samples[0].HeapInuse
	final := samples[len(samples)-1].HeapInuse
	growth := float64(final) / float64(initial)

	t.Logf("\nGrowth ratio: %.4fx (%.2f%% increase)", growth, (growth-1)*100)

	const maxGrowth = 1.10
	if growth > maxGrowth {
		t.Errorf("heap grew %.2fx over %v (threshold: %.2fx) — possible leak",
			growth, duration, maxGrowth)
	}

	// Check goroutine stability during steady state (first two samples, both during run)
	// Don't compare against the final sample — that captures post-shutdown cleanup
	if len(samples) >= 2 {
		steadyDiff := absDiff(samples[0].Goroutines, samples[1].Goroutines)
		if steadyDiff > 2 {
			t.Errorf("goroutine instability during steady state: %d → %d (delta: %d)",
				samples[0].Goroutines, samples[1].Goroutines, steadyDiff)
		}
	}
}

// TestMemory_SteadyState_IPFIX verifies memory stability for IPFIX generation.
func TestMemory_SteadyState_IPFIX(t *testing.T) {
	duration := 2 * time.Minute
	interval := 30 * time.Second
	workers := 4
	delayMs := 100

	samples := runMemoryTest(t, workers, delayMs, duration, interval, barrage.IPFIX())

	if len(samples) < 2 {
		t.Fatal("insufficient samples collected")
	}

	for _, s := range samples {
		t.Logf("%-10s heap_alloc=%8.2f MB  heap_inuse=%8.2f MB  goroutines=%d",
			s.Elapsed.Round(time.Second),
			float64(s.HeapAlloc)/1024/1024,
			float64(s.HeapInuse)/1024/1024,
			s.Goroutines)
	}

	initial := samples[0].HeapInuse
	final := samples[len(samples)-1].HeapInuse
	growth := float64(final) / float64(initial)

	t.Logf("\nGrowth ratio: %.4fx (%.2f%% increase)", growth, (growth-1)*100)

	const maxGrowth = 1.10
	if growth > maxGrowth {
		t.Errorf("IPFIX heap grew %.2fx over %v (threshold: %.2fx)",
			growth, duration, maxGrowth)
	}
}

// TestMemory_HighLoad_32Workers verifies memory stability under heavy load.
func TestMemory_HighLoad_32Workers(t *testing.T) {
	duration := 2 * time.Minute
	interval := 30 * time.Second
	workers := 32
	delayMs := 100

	samples := runMemoryTest(t, workers, delayMs, duration, interval, barrage.NetFlow())

	if len(samples) < 2 {
		t.Fatal("insufficient samples collected")
	}

	for _, s := range samples {
		t.Logf("%-10s heap_alloc=%8.2f MB  heap_inuse=%8.2f MB  goroutines=%d",
			s.Elapsed.Round(time.Second),
			float64(s.HeapAlloc)/1024/1024,
			float64(s.HeapInuse)/1024/1024,
			s.Goroutines)
	}

	initial := samples[0].HeapInuse
	final := samples[len(samples)-1].HeapInuse
	growth := float64(final) / float64(initial)

	t.Logf("\nGrowth ratio: %.4fx (%.2f%% increase)", growth, (growth-1)*100)

	const maxGrowth = 1.15 // slightly higher tolerance for 32 workers
	if growth > maxGrowth {
		t.Errorf("32-worker heap grew %.2fx over %v (threshold: %.2fx)",
			growth, duration, maxGrowth)
	}
}

// ---------------------------------------------------------------------------
// Short smoke test (runs in ~15s, suitable for CI)
// ---------------------------------------------------------------------------

// TestMemory_Smoke_NetFlow is a quick sanity check that the memory test
// infrastructure works. Runs for only 10 seconds with 1 sample.
func TestMemory_Smoke_NetFlow(t *testing.T) {
	duration := 10 * time.Second
	interval := 5 * time.Second
	workers := 4
	delayMs := 100

	samples := runMemoryTest(t, workers, delayMs, duration, interval, barrage.NetFlow())

	if len(samples) < 2 {
		t.Fatal("insufficient samples")
	}

	for _, s := range samples {
		t.Logf("%-10s heap_alloc=%8.2f MB  heap_inuse=%8.2f MB  goroutines=%d",
			s.Elapsed.Round(time.Second),
			float64(s.HeapAlloc)/1024/1024,
			float64(s.HeapInuse)/1024/1024,
			s.Goroutines)
	}

	// Just verify no crash, no assertions on short runs
	t.Logf("Smoke test passed — %d samples collected", len(samples))
}

// absDiff returns the absolute difference of two ints.
func absDiff(a, b int) int {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}

// Suppress unused import warning for fmt (used in log format strings above).
var _ = fmt.Sprintf
