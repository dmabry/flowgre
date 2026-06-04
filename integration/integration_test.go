//go:build integration

// Integration tests for the full record→replay pipeline.
// Exercises real UDP sockets, BadgerDB storage, and binary round-trips.
//
// Run with: go test -tags=integration -v ./integration
package integration

import (
	"context"
	"encoding/binary"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/record"
	"github.com/dmabry/flowgre/single"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func pickPort(t *testing.T) int {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("pickPort: %v", err)
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	conn.Close()
	return port
}

func countDBEntries(t *testing.T, dir string) (total, v9, v10 int) {
	t.Helper()
	db, err := badger.Open(badger.DefaultOptions(dir).WithReadOnly(true))
	if err != nil {
		t.Fatalf("countDBEntries: %v", err)
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
		t.Fatal(err)
	}
	return
}

func startRecorder(ctx context.Context, t *testing.T, ip string, port int, dbDir string) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		record.RunCtx(ctx, ip, port, dbDir, false)
		close(done)
	}()
	time.Sleep(500 * time.Millisecond)
	return func() { cancel(); <-done }
}

func sendRawPacket(t *testing.T, port int, data []byte) {
	t.Helper()
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatalf("sendRawPacket: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("sendRawPacket write: %v", err)
	}
}

// sendIPFIX sends IPFIX packets directly using the ipfix package.
func sendIPFIX(t *testing.T, port int, count int) {
	t.Helper()
	session := netflow.NewSession()
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatalf("sendIPFIX dial: %v", err)
	}
	defer conn.Close()

	// Send template
	tpl := ipfix.GenerateTemplateIPFIX(1, session)
	tplBuf := tpl.ToBytes()
	conn.Write(tplBuf.Bytes())

	// Send data flows
	for i := 0; i < count; i++ {
		pkt, err := ipfix.GenerateDataIPFIX(1, 1, "10.0.0.0/8", "172.16.0.0/12", 0, session)
		if err != nil {
			t.Fatal(err)
		}
		pktBuf := pkt.ToBytes()
		conn.Write(pktBuf.Bytes())
		time.Sleep(10 * time.Millisecond)
	}
}

// replayFromDB reads packets from a BadgerDB and sends them to the given port.
func replayFromDB(t *testing.T, dbDir string, port int) {
	t.Helper()
	db, err := badger.Open(badger.DefaultOptions(dbDir).WithReadOnly(true))
	if err != nil {
		t.Fatalf("replayFromDB open: %v", err)
	}
	defer db.Close()

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatalf("replayFromDB dial: %v", err)
	}
	defer conn.Close()

	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			val, err := it.Item().ValueCopy(nil)
			if err != nil {
				continue
			}
			conn.Write(val)
			time.Sleep(10 * time.Millisecond)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("replayFromDB iterate: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRoundTrip_NetFlow(t *testing.T) {
	t.Parallel()
	tmpDir := mustTempDir(t)
	defer os.RemoveAll(tmpDir)

	recPort := pickPort(t)
	replayPort := pickPort(t)
	recDB := filepath.Join(tmpDir, "rec")
	replayDB := filepath.Join(tmpDir, "replay")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopRec := startRecorder(ctx, t, "127.0.0.1", recPort, recDB)
	stopReplay := startRecorder(ctx, t, "127.0.0.1", replayPort, replayDB)

	// Generate NetFlow v9 via single.RunCtx
	sCtx, sCancel := context.WithCancel(context.Background())
	sDone := make(chan struct{})
	go func() {
		single.RunCtx(sCtx, "127.0.0.1", recPort, 0, 10, "10.0.0.0/8", "172.16.0.0/12", false)
		close(sDone)
	}()
	// Wait for sender to finish (it sends 10 flows then exits)
	select {
	case <-sDone:
	case <-time.After(10 * time.Second):
		sCancel()
	}
	sCancel()

	time.Sleep(300 * time.Millisecond)
	stopRec()
	stopReplay()
	cancel()

	total, v9, v10 := countDBEntries(t, recDB)
	if v9 != 11 {
		t.Errorf("NetFlow record: expected 11 v9 packets, got %d (total=%d, v10=%d)", v9, total, v10)
	}
	if v10 != 0 {
		t.Errorf("NetFlow record: expected 0 v10 packets, got %d", v10)
	}

	// Replay through second recorder
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	stopReplay2 := startRecorder(ctx2, t, "127.0.0.1", replayPort, replayDB)

	replayFromDB(t, recDB, replayPort)
	time.Sleep(300 * time.Millisecond)
	stopReplay2()
	cancel2()

	rtTotal, rtV9, rtV10 := countDBEntries(t, replayDB)
	if rtV9 != 11 {
		t.Errorf("NetFlow replay: expected 11 v9 packets, got %d (total=%d, v10=%d)", rtV9, rtTotal, rtV10)
	}
}

func TestRoundTrip_IPFIX(t *testing.T) {
	t.Parallel()
	tmpDir := mustTempDir(t)
	defer os.RemoveAll(tmpDir)

	recPort := pickPort(t)
	replayPort := pickPort(t)
	recDB := filepath.Join(tmpDir, "rec")
	replayDB := filepath.Join(tmpDir, "replay")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopRec := startRecorder(ctx, t, "127.0.0.1", recPort, recDB)
	stopReplay := startRecorder(ctx, t, "127.0.0.1", replayPort, replayDB)

	// Generate IPFIX via direct package API
	sendIPFIX(t, recPort, 10)
	time.Sleep(300 * time.Millisecond)

	stopRec()
	stopReplay()
	cancel()

	total, v9, v10 := countDBEntries(t, recDB)
	if v10 != 11 {
		t.Errorf("IPFIX record: expected 11 v10 packets, got %d (total=%d, v9=%d)", v10, total, v9)
	}
	if v9 != 0 {
		t.Errorf("IPFIX record: expected 0 v9 packets, got %d", v9)
	}

	// Replay
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	stopReplay2 := startRecorder(ctx2, t, "127.0.0.1", replayPort, replayDB)

	replayFromDB(t, recDB, replayPort)
	time.Sleep(300 * time.Millisecond)
	stopReplay2()
	cancel2()

	rtTotal, rtV9, rtV10 := countDBEntries(t, replayDB)
	if rtV10 != 11 {
		t.Errorf("IPFIX replay: expected 11 v10 packets, got %d (total=%d, v9=%d)", rtV10, rtTotal, rtV9)
	}
}

func TestNegative_GarbageRejected(t *testing.T) {
	t.Parallel()
	tmpDir := mustTempDir(t)
	defer os.RemoveAll(tmpDir)

	port := pickPort(t)
	dbDir := filepath.Join(tmpDir, "neg")

	ctx, cancel := context.WithCancel(context.Background())
	stop := startRecorder(ctx, t, "127.0.0.1", port, dbDir)

	tests := []struct {
		label string
		data  []byte
	}{
		{"random-256", makeRandom(256)},
		{"ver-99", makeFakeHeader(99, 20)},
		{"tiny-2", []byte{0xFF, 0xFE}},
		{"empty-4", make([]byte, 4)},
		{"dns-like", makeFakeHeader(0xABCD, 60)},
		{"fake-nfv9", makeFakeHeader(9, 100)},
		{"fake-ipfix", makeFakeHeader(10, 128)},
		{"seq-noise", makeSequential(512)},
		{"icmp-like", makeFakeHeader(0x0800, 64)},
		{"all-zeros", make([]byte, 100)},
	}

	for _, tt := range tests {
		sendRawPacket(t, port, tt.data)
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)
	cancel()
	stop()

	total, v9, v10 := countDBEntries(t, dbDir)
	const wantTotal = 2
	const wantV9 = 1
	const wantV10 = 1

	if total != wantTotal {
		t.Errorf("stored %d packets, expected %d", total, wantTotal)
		for i, tt := range tests {
			t.Logf("  [%d] %-15s (%d bytes)", i, tt.label, len(tt.data))
		}
	}
	if v9 != wantV9 {
		t.Errorf("v9 count: got %d, want %d", v9, wantV9)
	}
	if v10 != wantV10 {
		t.Errorf("v10 count: got %d, want %d", v10, wantV10)
	}
}

func TestValidation_Unit(t *testing.T) {
	t.Parallel()

	if ok, err := netflow.IsValidNetFlow([]byte{9}, 9); ok || err == nil {
		t.Error("NetFlow: too-short payload should error")
	}
	if ok, err := ipfix.IsValidIPFIX([]byte{10}); ok || err == nil {
		t.Error("IPFIX: too-short payload should error")
	}

	wrongNF := makeFakeHeader(10, 20)
	ok, err := netflow.IsValidNetFlow(wrongNF, 9)
	if ok || err == nil {
		t.Error("NetFlow: version mismatch should fail")
	}

	wrongIPFIX := makeFakeHeader(9, 20)
	ok, err = ipfix.IsValidIPFIX(wrongIPFIX)
	if ok || err == nil {
		t.Error("IPFIX: version mismatch should fail")
	}

	validNF := makeFakeHeader(9, 100)
	ok, err = netflow.IsValidNetFlow(validNF, 9)
	if !ok || err != nil {
		t.Errorf("NetFlow: valid v9 should pass: ok=%v, err=%v", ok, err)
	}

	validIPFIX := makeFakeHeader(10, 128)
	ok, err = ipfix.IsValidIPFIX(validIPFIX)
	if !ok || err != nil {
		t.Errorf("IPFIX: valid v10 should pass: ok=%v, err=%v", ok, err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "flowgre-int-*")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func makeFakeHeader(version uint16, size int) []byte {
	buf := make([]byte, size)
	binary.BigEndian.PutUint16(buf[:2], version)
	binary.BigEndian.PutUint32(buf[4:8], 1)
	return buf
}

func makeRandom(size int) []byte {
	buf := make([]byte, size)
	rand.Read(buf)
	return buf
}

func makeSequential(size int) []byte {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte(i)
	}
	return buf
}
