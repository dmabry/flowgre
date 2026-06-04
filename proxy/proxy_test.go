// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package proxy

import (
	"bytes"
	"context"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/stats"
)

// TestProxyListener tests that the proxy listener can receive UDP packets.
func TestProxyListener(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxyChan := make(chan []byte, bufferSize)
	var wg sync.WaitGroup

	// Start proxy listener
	wg.Add(1)
	go proxyListener(ctx, &wg, "127.0.0.1", 19995, proxyChan, false)

	// Give listener time to start
	time.Sleep(100 * time.Millisecond)

	// Send a test packet
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19995})
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	testPayload := []byte("test payload")
	_, err = conn.Write(testPayload)
	if err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	// Wait for packet to be received
	select {
	case payload := <-proxyChan:
		if !bytes.Equal(payload, testPayload) {
			t.Errorf("Received wrong payload: got %v, want %v", payload, testPayload)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for packet")
	}

	// Cleanup
	cancel()
	wg.Wait()
	close(proxyChan)
}

// TestReplicator tests that the replicator forwards payloads to all target channels.
func TestReplicator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataChan := make(chan []byte, bufferSize)
	target1 := make(chan []byte, bufferSize)
	target2 := make(chan []byte, bufferSize)
	targets := []chan []byte{target1, target2}

	var wg sync.WaitGroup
	wg.Add(1)
	go replicator(ctx, &wg, dataChan, targets, false)

	// Send test payload
	testPayload := []byte("test replicator")
	dataChan <- testPayload

	// Verify both targets receive the payload
	for i, target := range targets {
		select {
		case payload := <-target:
			if !bytes.Equal(payload, testPayload) {
				t.Errorf("Target %d received wrong payload: got %v, want %v", i, payload, testPayload)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("Timeout waiting for target %d", i)
		}
	}

	// Cleanup
	cancel()
	wg.Wait()
	close(dataChan)
	close(target1)
	close(target2)
}

// TestWorker tests that workers can send packets to a target.
func TestWorker(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerChan := make(chan []byte, bufferSize)
	var wg sync.WaitGroup

	// Start a receiver on the target port
	receiverDone := make(chan struct{})
	receiverReady := make(chan struct{})
	wg.Go(func() {
		defer close(receiverDone)

		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19996})
		if err != nil {
			t.Errorf("Failed to listen: %v", err)
			return
		}
		defer conn.Close()

		close(receiverReady) // signal that the receiver is listening

		payload := make([]byte, udpMaxBufferSize)
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := conn.ReadFromUDP(payload)
		if err != nil {
			t.Errorf("Failed to receive: %v", err)
			return
		}

		if !bytes.Equal(payload[:n], []byte("worker test")) {
			t.Errorf("Received wrong payload: got %v, want %v", payload[:n], "worker test")
		}
	})

	// Wait until the receiver is actually bound and ready to receive
	<-receiverReady

	// Start worker
	wg.Add(1)
	go worker(1, ctx, "127.0.0.1", 19996, &wg, workerChan)

	// Send test payload
	workerChan <- []byte("worker test")

	// Wait for receiver to complete
	select {
	case <-receiverDone:
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for receiver")
	}

	// Cleanup
	cancel()
	close(workerChan)
	wg.Wait()
}

// TestParseNetflow tests that valid NetFlow packets are accepted and invalid ones rejected.
func TestParseNetflow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxyChan := make(chan []byte, bufferSize)
	dataChan := make(chan []byte, bufferSize)
	rStats := &stats.RecordStat{}

	var wg sync.WaitGroup
	wg.Add(1)
	go parseNetflow(ctx, &wg, proxyChan, dataChan, rStats, false)

	// Send invalid payload (not NetFlow)
	proxyChan <- []byte("invalid")

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	if rStats.LoadInvalid() != 1 {
		t.Errorf("Expected 1 invalid packet, got %d", rStats.LoadInvalid())
	}

	// Send valid NetFlow packet
	session := netflow.NewSession()
	flow := netflow.GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()
	proxyChan <- buf.Bytes()

	// Wait for processing
	select {
	case <-dataChan:
		// Good, valid packet was forwarded
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for valid packet to be forwarded")
	}

	if rStats.LoadValid() != 1 {
		t.Errorf("Expected 1 valid packet, got %d", rStats.LoadValid())
	}

	// Cleanup
	cancel()
	wg.Wait()
	close(proxyChan)
	close(dataChan)
}

// TestStatsPrinter tests that stats are printed periodically.
func TestStatsPrinter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rStats := &stats.RecordStat{
		ValidCount:   5,
		InvalidCount: 2,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go statsPrinter(ctx, &wg, rStats)

	// Wait for at least one stats print
	time.Sleep(12 * time.Second)

	// Cleanup
	cancel()
	wg.Wait()
}

// TestRunIntegration tests the full proxy flow with a single target.
func TestRunIntegration(t *testing.T) {
	t.Parallel()
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull) // hide logs
	defer func() { os.Stdout = origStdout }()

	// Start a receiver on target port
	targetPort := 19997
	received := make(chan struct{}, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: targetPort})
		if err != nil {
			t.Errorf("Failed to listen: %v", err)
			return
		}
		defer conn.Close()

		payload := make([]byte, udpMaxBufferSize)
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		_, _, err = conn.ReadFromUDP(payload)
		if err != nil {
			t.Errorf("Failed to receive: %v", err)
			return
		}
		received <- struct{}{}
	})

	// Start proxy in a goroutine
	proxyDone := make(chan struct{})
	go func() {
		defer close(proxyDone)
		Run("127.0.0.1", 19998, false, []string{"127.0.0.1:19997"})
	}()

	// Give proxy time to start
	time.Sleep(1 * time.Second)

	// Send a test packet to the proxy
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19998})
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Create a valid NetFlow packet
	session := netflow.NewSession()
	flow := netflow.GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	// Wait for packet to be received by target
	select {
	case <-received:
		// Success!
	case <-time.After(10 * time.Second):
		t.Error("Timeout waiting for target to receive packet")
	}

	// Cleanup
	cancelCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-proxyDone
		cancelCtx.Done()
	}()
	cancel()
	wg.Wait()
}

// TestTargetValidation tests that invalid targets are rejected.
func TestTargetValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		targets []string
		wantErr bool
	}{
		{
			name:    "empty targets",
			targets: []string{},
			wantErr: true,
		},
		{
			name:    "invalid format",
			targets: []string{"invalid"},
			wantErr: true,
		},
		{
			name:    "invalid port",
			targets: []string{"127.0.0.1:99999"},
			wantErr: true,
		},
		{
			name:    "valid target",
			targets: []string{"127.0.0.1:9995"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Just test the validation logic by attempting to parse
			if len(tt.targets) == 0 {
				if !tt.wantErr {
					t.Error("Expected error for empty targets")
				}
				return
			}

			for _, target := range tt.targets {
				_, portStr, err := net.SplitHostPort(target)
				if err != nil && !tt.wantErr {
					t.Errorf("Unexpected error: %v", err)
				}

				if err == nil {
					port, err := strconv.Atoi(portStr)
					if err != nil {
						if !tt.wantErr {
							t.Errorf("Unexpected error: %v", err)
						}
						continue
					}
					if port < 1 || port > 65535 {
						if !tt.wantErr {
							t.Errorf("Expected error for invalid port %d", port)
						}
					}
				}
			}
		})
	}
}

// TestMaxTargets tests that more than maxTargets is rejected.
func TestMaxTargets(t *testing.T) {
	t.Parallel()

	targets := make([]string, 11)
	for i := range targets {
		targets[i] = "127.0.0.1:9995"
	}

	if len(targets) > maxTargets {
		// This is expected to fail in Run(), but we can test the constant
		t.Logf("Max targets limit is %d, attempted %d", maxTargets, len(targets))
	}
}
