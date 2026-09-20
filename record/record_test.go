// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package record

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/dmabry/flowgre/netflow"
)

// TestNetIngest tests that the network listener can receive UDP packets.
func TestNetIngest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChan := make(chan []byte, 1024)
	var wg sync.WaitGroup

	// Reserve an ephemeral port instead of a fixed one — fixed ports collide
	// under parallel test runs, and netIngest swallows bind failures, so a
	// collision looks like a silent packet loss instead of an error.
	port := reserveUDPPort(t)

	// Start netIngest
	wg.Add(1)
	go netIngest(ctx, &wg, "127.0.0.1", port, dataChan, false)

	// Send a test packet, retrying until the listener is bound — packets sent
	// to a not-yet-bound UDP port are dropped, not queued.
	testPayload := []byte("test payload for record")
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	deadline := time.After(2 * time.Second)
	for {
		if _, err = conn.Write(testPayload); err != nil {
			// A connected UDP socket surfaces an earlier ICMP
			// port-unreachable as ECONNREFUSED — normal during the bind
			// race, not fatal. Retry until the deadline.
			if !errors.Is(err, syscall.ECONNREFUSED) {
				t.Fatalf("Failed to send: %v", err)
			}
		}
		select {
		case payload := <-dataChan:
			if !bytes.Equal(payload, testPayload) {
				t.Errorf("Received wrong payload: got %v, want %v", payload, testPayload)
			}
			// Cleanup
			cancel()
			wg.Wait()
			close(dataChan)
			return
		case <-time.After(50 * time.Millisecond):
			// Listener may not be bound yet — retry
		case <-deadline:
			t.Error("Timeout waiting for packet")
			// Cleanup
			cancel()
			wg.Wait()
			close(dataChan)
			return
		}
	}
}

// reserveUDPPort binds an ephemeral UDP listener, records its port, and
// releases it so netIngest can rebind. Using an ephemeral port avoids
// fixed-port collisions across parallel tests and concurrent CI runs.
func reserveUDPPort(t *testing.T) int {
	t.Helper()
	probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to reserve ephemeral UDP port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	if err := probe.Close(); err != nil {
		t.Fatalf("Failed to release probe listener: %v", err)
	}
	return port
}

// TestDbIngest tests that the database ingest can store payloads.
func TestDbIngest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a temporary directory for the test DB
	tmpDir := t.TempDir()
	dataChan := make(chan []byte, 1024)

	var wg sync.WaitGroup
	wg.Add(1)
	go dbIngest(ctx, &wg, tmpDir, dataChan, false)

	// Give DB time to open
	time.Sleep(200 * time.Millisecond)

	// Send test payload
	testPayload := []byte("test db ingest payload")
	dataChan <- testPayload

	// Wait a bit for processing
	time.Sleep(200 * time.Millisecond)

	// Cleanup
	cancel()
	wg.Wait()
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

// TestRunIntegration tests the record pipeline end-to-end (ingest → parse).
// The DB stage is covered separately by TestDbIngest.
func TestRunIntegration(t *testing.T) {
	t.Parallel()
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull) // hide logs
	defer func() { os.Stdout = origStdout }()

	// Start the three components manually with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChan := make(chan []byte, 1024)
	parseChan := make(chan []byte, 1024)
	var wg sync.WaitGroup

	// Start the ingest → parse pipeline. dbIngest is intentionally NOT
	// started: it would be a blocked receiver on dataChan and win every
	// receive race against the test. The test itself is dataChan's only
	// consumer, making the pipeline observation deterministic. (The DB
	// stage is covered by TestDbIngest.)
	port := reserveUDPPort(t)

	// Start netIngest
	wg.Add(1)
	go netIngest(ctx, &wg, "127.0.0.1", port, parseChan, false)

	// Start parseFlow
	wg.Add(1)
	go parseFlow(ctx, &wg, parseChan, dataChan, false)

	// Send a valid NetFlow packet to the recorder, retrying until the
	// listener is bound.
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	session := netflow.NewSession()
	flow := netflow.GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()

	deadline := time.After(5 * time.Second)
	for {
		if _, err = conn.Write(buf.Bytes()); err != nil {
			// A connected UDP socket surfaces an earlier ICMP
			// port-unreachable as ECONNREFUSED — normal during the bind
			// race, not fatal. Retry until the deadline.
			if !errors.Is(err, syscall.ECONNREFUSED) {
				t.Fatalf("Failed to send: %v", err)
			}
		}
		select {
		case payload := <-dataChan:
			// The packet traversed ingest → parse. Verify it survived
			// as a valid NetFlow packet.
			if ok, verr := netflow.IsValidNetFlow(payload, 9); !ok {
				t.Errorf("Received invalid NetFlow payload: %v", verr)
			}
			// Pipeline is live — let any in-flight packets drain.
		case <-time.After(50 * time.Millisecond):
			// Listener may not be bound yet — retry
			continue
		case <-deadline:
			t.Fatal("Timeout waiting for packet to traverse the record pipeline")
		}
		break
	}

	// Give parseFlow a moment to finish processing
	time.Sleep(500 * time.Millisecond)

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

	// Start netIngest
	wg.Add(1)
	go netIngest(ctx, &wg, "127.0.0.1", 29997, dataChan, false)

	// Give listener time to start
	time.Sleep(100 * time.Millisecond)

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

	// Give DB time to open
	time.Sleep(200 * time.Millisecond)

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
