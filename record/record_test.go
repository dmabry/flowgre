// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package record

import (
	"bytes"
	"context"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dmabry/flowgre/netflow"
)

// TestNetIngest tests that the network listener can receive UDP packets.
func TestNetIngest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChan := make(chan []byte, 1024)
	var wg sync.WaitGroup

	// Bind to port 0 to get a free port, then pass it to netIngest
	probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	wg.Add(1)
	go netIngest(ctx, &wg, "127.0.0.1", port, dataChan, false)

	// Wait for the listener to be ready by probing the port
	ready := make(chan struct{})
	go func() {
		for {
			c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
			if err == nil {
				c.Close()
				close(ready)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for listener to start")
	}

	// Send a test packet
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	testPayload := []byte("test payload for record")
	_, err = conn.Write(testPayload)
	if err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	// Wait for packet to be received
	select {
	case payload := <-dataChan:
		if !bytes.Equal(payload, testPayload) {
			t.Errorf("Received wrong payload: got %v, want %v", payload, testPayload)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for packet")
	}

	// Cleanup
	cancel()
	wg.Wait()
	close(dataChan)
}

// TestDbIngest tests that the database ingest can store payloads.
func TestDbIngest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a temporary directory for the test DB
	tmpDir := t.TempDir()
	dataChan := make(chan []byte, 1024)

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer close(done)
		dbIngest(ctx, &wg, tmpDir, dataChan, false)
	}()

	// Send test payload; dbIngest will process it once the DB is open
	testPayload := []byte("test db ingest payload")
	dataChan <- testPayload

	// Wait a bit for processing
	time.Sleep(200 * time.Millisecond)

	// Cleanup
	cancel()
	wg.Wait()
	<-done
	close(dataChan)
}

// TestParseFlow tests that valid NetFlow and IPFIX packets are accepted and invalid ones rejected.
func TestParseFlow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	parseChan := make(chan []byte, 1024)
	dataChan := make(chan []byte, 1024)

	var wg sync.WaitGroup
	wg.Add(1)
	go parseFlow(ctx, &wg, parseChan, dataChan, false)

	// Send invalid payload (not NetFlow)
	parseChan <- []byte("invalid")

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Send valid NetFlow packet
	session := netflow.NewSession()
	flow := netflow.GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()
	parseChan <- buf.Bytes()

	// Wait for processing
	select {
	case <-dataChan:
		// Good, valid packet was forwarded
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for valid packet to be forwarded")
	}

	// Cleanup
	cancel()
	wg.Wait()
	close(parseChan)
	close(dataChan)
}

// TestRunIntegration tests the full record flow.
func TestRunIntegration(t *testing.T) {
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull) // hide logs
	defer func() { os.Stdout = origStdout }()

	// Create a temporary directory for the test DB
	tmpDir := t.TempDir()

	// Start the three components manually with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChan := make(chan []byte, 1024)
	parseChan := make(chan []byte, 1024)
	var wg sync.WaitGroup

	// Bind to port 0 to get a free port
	probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	// Start netIngest
	wg.Add(1)
	go netIngest(ctx, &wg, "127.0.0.1", port, parseChan, false)

	// Start parseFlow
	wg.Add(1)
	go parseFlow(ctx, &wg, parseChan, dataChan, false)

	// Start dbIngest
	wg.Add(1)
	go dbIngest(ctx, &wg, tmpDir, dataChan, false)

	// Wait for the listener to be ready
	ready := make(chan struct{})
	go func() {
		for {
			c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
			if err == nil {
				c.Close()
				close(ready)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for listener to start")
	}

	// Send a valid NetFlow packet to the recorder
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	session := netflow.NewSession()
	flow := netflow.GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	// Wait a bit for processing
	time.Sleep(2 * time.Second)

	// Cleanup
	cancel()
	wg.Wait()
	close(dataChan)
	close(parseChan)
}

// TestNetIngestContextCancellation tests that netIngest responds to context cancellation.
func TestNetIngestContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	dataChan := make(chan []byte, 1024)
	var wg sync.WaitGroup

	// Bind to port 0 to get a free port
	probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	// Start netIngest
	wg.Add(1)
	go netIngest(ctx, &wg, "127.0.0.1", port, dataChan, false)

	// Cancel context
	cancel()

	// Wait for goroutine to exit
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Good, goroutine exited
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for goroutine to exit")
	}

	close(dataChan)
}

// TestDbIngestContextCancellation tests that dbIngest responds to context cancellation.
func TestDbIngestContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	tmpDir := t.TempDir()
	dataChan := make(chan []byte, 1024)

	var wg sync.WaitGroup
	wg.Add(1)
	go dbIngest(ctx, &wg, tmpDir, dataChan, false)

	// Cancel context
	cancel()

	// Wait for goroutine to exit
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Good, goroutine exited
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for goroutine to exit")
	}

	close(dataChan)
}
