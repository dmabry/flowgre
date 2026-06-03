//go:build bench

// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package bench contains sustained throughput benchmarks for flowgre.
// These exercise real UDP sockets, BadgerDB storage, and the full barrage pipeline.
//
// Run with: go test -tags=bench -bench=BenchmarkThroughput -run=^$ ./bench/
package bench

import (
	"context"
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/dmabry/flowgre/barrage"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/record"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func pickPortB(b *testing.B) int {
	b.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		b.Fatalf("pickPort: %v", err)
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	conn.Close()
	return port
}

func countDBEntriesB(b *testing.B, dir string) (total, v9, v10 int) {
	b.Helper()
	db, err := badger.Open(badger.DefaultOptions(dir).WithReadOnly(true))
	if err != nil {
		b.Fatalf("countDBEntries: %v", err)
	}
	defer db.Close()

	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			val, err := it.Item().ValueCopy(nil)
			if err != nil || len(val) < 2 {
				continue
			}
			total++
			switch binary.BigEndian.Uint16(val[:2]) {
			case 9:
				v9++
			case 10:
				v10++
			}
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	return
}

// runThroughput runs a full barrage + recorder cycle and returns packets recorded.
func runThroughput(b *testing.B, workers, delayMs int, duration time.Duration, gen barrage.FlowGenerator) (total, v9, v10 int) {
	b.Helper()

	tmpDir, err := os.MkdirTemp("", "flowgre-bench-*")
	if err != nil {
		b.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	recPort := pickPortB(b)
	dbDir := filepath.Join(tmpDir, "rec")

	// Start recorder with dedicated context + waitgroup
	recCtx, recCancel := context.WithCancel(context.Background())
	var recWg sync.WaitGroup
	recWg.Add(1)
	go func() {
		defer recWg.Done()
		record.RunCtx(recCtx, "127.0.0.1", recPort, dbDir, false)
	}()
	time.Sleep(500 * time.Millisecond) // wait for recorder to bind

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
	var barrWg sync.WaitGroup
	barrWg.Add(1)
	go func() {
		defer barrWg.Done()
		barrage.RunCtx(barrCtx, cfg, gen)
	}()

	// Wait for barrage to finish
	barrWg.Wait()
	barrCancel()

	// Cancel recorder and wait for clean shutdown
	recCancel()
	recWg.Wait()

	// Extra pause to ensure BadgerDB file locks are released
	time.Sleep(1 * time.Second)

	return countDBEntriesB(b, dbDir)
}

// ---------------------------------------------------------------------------
// Worker Scaling (100ms delay, NetFlow)
// ---------------------------------------------------------------------------

// BenchmarkThroughput_1Worker_100ms measures 1 worker at 100ms delay.
func BenchmarkThroughput_1Worker_100ms(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 1, 100, 10*time.Second, barrage.NetFlow())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}

// BenchmarkThroughput_4Workers_100ms measures 4 workers at 100ms delay.
func BenchmarkThroughput_4Workers_100ms(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 4, 100, 10*time.Second, barrage.NetFlow())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}

// BenchmarkThroughput_8Workers_100ms measures 8 workers at 100ms delay.
func BenchmarkThroughput_8Workers_100ms(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 8, 100, 10*time.Second, barrage.NetFlow())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}

// BenchmarkThroughput_16Workers_100ms measures 16 workers at 100ms delay.
func BenchmarkThroughput_16Workers_100ms(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 16, 100, 10*time.Second, barrage.NetFlow())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}

// BenchmarkThroughput_32Workers_100ms measures 32 workers at 100ms delay.
func BenchmarkThroughput_32Workers_100ms(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 32, 100, 10*time.Second, barrage.NetFlow())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}

// ---------------------------------------------------------------------------
// Delay Sensitivity (4 workers, NetFlow)
// ---------------------------------------------------------------------------

// BenchmarkThroughput_4Workers_10ms measures 4 workers at 10ms delay.
func BenchmarkThroughput_4Workers_10ms(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 4, 10, 10*time.Second, barrage.NetFlow())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}

// BenchmarkThroughput_4Workers_50ms measures 4 workers at 50ms delay.
func BenchmarkThroughput_4Workers_50ms(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 4, 50, 10*time.Second, barrage.NetFlow())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}

// BenchmarkThroughput_4Workers_200ms measures 4 workers at 200ms delay.
func BenchmarkThroughput_4Workers_200ms(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 4, 200, 10*time.Second, barrage.NetFlow())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}

// ---------------------------------------------------------------------------
// Protocol Comparison (4 workers, 100ms delay)
// ---------------------------------------------------------------------------

// BenchmarkThroughput_4Workers_IPFIX measures IPFIX throughput vs NetFlow.
func BenchmarkThroughput_4Workers_IPFIX(b *testing.B) {
	b.StopTimer()
	total, _, _ := runThroughput(b, 4, 100, 10*time.Second, barrage.IPFIX())
	b.StartTimer()
	b.ReportMetric(float64(total)/10, "pkt/s")
	b.ReportMetric(float64(total), "total_packets")
}
