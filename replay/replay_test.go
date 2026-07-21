// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package replay

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"os"
	"testing"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

// TestWorker tests that workers can send packets to a target.
func TestWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChan := make(chan []byte, 1024)
	receiverReady := make(chan struct{})

	// Bind to port 0 to get a free port
	probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	// Start a receiver on the target port
	received := make(chan struct{}, 1)
	go func() {
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
		if err != nil {
			t.Errorf("Failed to listen: %v", err)
			return
		}
		defer conn.Close()

		close(receiverReady) // signal that the receiver is listening

		payload := make([]byte, 65507)
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := conn.ReadFromUDP(payload)
		if err != nil {
			t.Errorf("Failed to receive: %v", err)
			return
		}

		if !bytes.Equal(payload[:n], []byte("worker test")) {
			t.Errorf("Received wrong payload: got %v, want %v", payload[:n], "worker test")
		}
		received <- struct{}{}
	}()

	// Wait until the receiver is actually bound and ready to receive
	<-receiverReady

	// Start worker
	done := make(chan struct{})
	go func() {
		worker(1, ctx, "127.0.0.1", port, 100, false, dataChan)
		close(done)
	}()

	// Send test payload
	dataChan <- []byte("worker test")

	// Wait for receiver to receive
	select {
	case <-received:
		// Success!
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for receiver")
	}

	// Cleanup
	cancel()
	close(dataChan)
	<-done
}

// TestDbReader tests that the database reader can read payloads from BadgerDB.
func TestDbReader(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a temporary directory for the test DB
	tmpDir := t.TempDir()
	dataChan := make(chan []byte, 1024)

	// First, write some test data to the DB
	options := badger.DefaultOptions(tmpDir)
	options.Logger = nil
	db, err := badger.Open(options)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	testPayload := []byte("test db reader payload")
	err = db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte("test-key"), testPayload)
		return txn.SetEntry(entry)
	})
	if err != nil {
		t.Fatalf("Failed to write to DB: %v", err)
	}
	db.Close()

	// Now read from the DB
	done := make(chan struct{})
	go func() {
		dbReader(ctx, tmpDir, dataChan, false, false, false)
		close(done)
	}()

	// Wait for data to be read
	select {
	case payload := <-dataChan:
		if !bytes.Equal(payload, testPayload) {
			t.Errorf("Received wrong payload: got %v, want %v", payload, testPayload)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for payload")
	}

	// Cleanup
	cancel()
	<-done
	// dataChan is closed by dbReader in non-loop mode
}

// TestDbReaderContextCancellation tests that dbReader responds to context cancellation.
func TestDbReaderContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	tmpDir := t.TempDir()
	dataChan := make(chan []byte, 1024)

	done := make(chan struct{})
	go func() {
		dbReader(ctx, tmpDir, dataChan, false, false, false)
		close(done)
	}()

	// Cancel context
	cancel()

	// Wait for goroutine to exit
	select {
	case <-done:
		// Good, goroutine exited
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for goroutine to exit")
	}

	// dataChan is closed by dbReader in non-loop mode, so don't close it again
}

// TestWorkerContextCancellation tests that worker responds to context cancellation.
func TestWorkerContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	dataChan := make(chan []byte, 1024)

	// Bind to port 0 to get a free port
	probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	// Start worker
	done := make(chan struct{})
	go func() {
		worker(1, ctx, "127.0.0.1", port, 100, false, dataChan)
		close(done)
	}()

	// Cancel context
	cancel()

	// Wait for goroutine to exit
	select {
	case <-done:
		// Good, goroutine exited
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for goroutine to exit")
	}

	close(dataChan)
}

func TestWorkerCancellationInterruptsRateLimitWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dataChan := make(chan []byte, 1)
	dataChan <- []byte("payload")

	done := make(chan error, 1)
	go func() {
		done <- worker(1, ctx, "127.0.0.1", 9995, 10_000, false, dataChan)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("worker returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not interrupt rate limit wait after cancellation")
	}
}

// TestRunIntegration tests the full replay flow.
func TestRunIntegration(t *testing.T) {
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull) // hide logs
	defer func() { os.Stdout = origStdout }()

	// Create a temporary directory for the test DB
	tmpDir := t.TempDir()

	// Write test data to DB
	options := badger.DefaultOptions(tmpDir)
	options.Logger = nil
	db, err := badger.Open(options)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	// Create a valid NetFlow packet to store
	session := netflow.NewSession()
	flow := netflow.GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()

	err = db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte("test-key"), buf.Bytes())
		return txn.SetEntry(entry)
	})
	if err != nil {
		t.Fatalf("Failed to write to DB: %v", err)
	}
	db.Close()

	// Bind to port 0 to get a free port
	probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	// Start a receiver on target port
	received := make(chan struct{}, 1)
	go func() {
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
		if err != nil {
			t.Errorf("Failed to listen: %v", err)
			return
		}
		defer conn.Close()

		payload := make([]byte, 65507)
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		_, _, err = conn.ReadFromUDP(payload)
		if err != nil {
			t.Errorf("Failed to receive: %v", err)
			return
		}
		received <- struct{}{}
	}()

	// Start replay
	replayDone := make(chan struct{})
	go func() {
		defer close(replayDone)
		Run("127.0.0.1", port, 100, tmpDir, false, 1, false, false)
	}()

	// Wait for packet to be received
	select {
	case <-received:
		// Success!
	case <-time.After(10 * time.Second):
		t.Error("Timeout waiting for target to receive packet")
	}

	// Wait for replay to complete
	<-replayDone
}

// TestSendPacket verifies that SendPacket works correctly.
func TestSendPacket(t *testing.T) {
	// Start a receiver
	received := make(chan []byte, 1)
	receiverReady := make(chan struct{})

	// Bind to port 0 to get a free port
	probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	go func() {
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
		if err != nil {
			t.Errorf("Failed to listen: %v", err)
			return
		}
		defer conn.Close()

		close(receiverReady) // signal that the receiver is listening

		payload := make([]byte, 65507)
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := conn.ReadFromUDP(payload)
		if err != nil {
			t.Errorf("Failed to receive: %v", err)
			return
		}
		received <- payload[:n]
	}()

	// Wait until the receiver is actually bound and ready to receive
	<-receiverReady

	// Send a packet
	srcPort, err := utils.RandomNum(10000, 15000)
	if err != nil {
		t.Fatalf("RandomNum error: %v", err)
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer conn.Close()

	testPayload := []byte("test send packet")

	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}, testPayload, false)
	if err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	// Wait for receiver to receive
	select {
	case payload := <-received:
		if !bytes.Equal(payload, testPayload) {
			t.Errorf("Received wrong payload: got %v, want %v", payload, testPayload)
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for receiver")
	}
}

// TestUpdateTimestampNetFlow tests that updateTimestamp correctly updates NetFlow v9 timestamps.
func TestUpdateTimestampNetFlow(t *testing.T) {
	t.Parallel()

	session := netflow.NewSession()
	flow := netflow.GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()
	payload := buf.Bytes()

	before := uint32(time.Now().Unix())
	result, err := updateTimestamp(payload)
	if err != nil {
		t.Fatalf("updateTimestamp failed: %v", err)
	}
	after := uint32(time.Now().Unix())

	// Verify the timestamp was updated (bytes 8-11 for NetFlow v9)
	newTS := binary.BigEndian.Uint32(result[8:12])
	if newTS < before || newTS > after {
		t.Errorf("NetFlow timestamp not updated: got %d, expected in [%d, %d]", newTS, before, after)
	}

	// Verify other fields preserved
	if binary.BigEndian.Uint16(result[0:2]) != 9 {
		t.Error("Version field corrupted")
	}
}

// TestUpdateTimestampIPFIX tests that updateTimestamp correctly updates IPFIX timestamps.
func TestUpdateTimestampIPFIX(t *testing.T) {
	t.Parallel()

	seq := ipfix.NewIPFIXSequence()
	ipfixPkt := ipfix.GenerateTemplateIPFIX(100, seq)
	payload, err := ipfixPkt.ToBytes()
	if err != nil {
		t.Fatalf("ToBytes failed: %v", err)
	}
	payloadBytes := payload.Bytes()

	before := uint32(time.Now().Unix())
	result, err := updateTimestamp(payloadBytes)
	if err != nil {
		t.Fatalf("updateTimestamp failed: %v", err)
	}
	after := uint32(time.Now().Unix())

	// Verify the Export Time was updated (bytes 4-7 for IPFIX)
	newTS := binary.BigEndian.Uint32(result[4:8])
	if newTS < before || newTS > after {
		t.Errorf("IPFIX Export Time not updated: got %d, expected in [%d, %d]", newTS, before, after)
	}

	// Verify Sequence Number preserved (bytes 8-11)
	origSeqNum := binary.BigEndian.Uint32(payloadBytes[8:12])
	newSeqNum := binary.BigEndian.Uint32(result[8:12])
	if origSeqNum != newSeqNum {
		t.Errorf("IPFIX Sequence Number corrupted: got %d, want %d", newSeqNum, origSeqNum)
	}
}
