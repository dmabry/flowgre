// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package barrage

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/models"
)

func ipfixReceiver(ctx context.Context, wg *sync.WaitGroup, ip string, port int, t *testing.T) {
	defer wg.Done()
	parsedFlows := 0

	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		t.Errorf("Test Receiver listening on %s:%d failed! Got: %v", ip, port, err)
		return
	}
	t.Logf("Test Receiver listening on %s:%d", ip, port)
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Errorf("Error closing listener: %v", closeErr)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			t.Log("Test Receiver exiting due to signal")
			t.Logf("Parsed IPFIX flows: %d", parsedFlows)
			return
		default:
			payload := make([]byte, udpMaxBufferSize)
			timeout := time.Now().Add(5 * time.Second)
			err := conn.SetReadDeadline(timeout)
			if err != nil {
				t.Errorf("Issue setting deadline: %v", err)
				return
			}
			length, _, err := conn.ReadFromUDP(payload)
			if err != nil {
				continue
			}
			payload = payload[:length]
			ok, err := ipfix.IsValidIPFIX(payload)
			if err != nil || !ok {
				t.Errorf("Invalid IPFIX Packet: %v", err)
				return
			}
			parsedFlows++
		}
	}
}

func ipfixRunWrapper(ctx context.Context, wg *sync.WaitGroup, duration int, bconfig *models.Config) {
	defer wg.Done()
	go RunIPFIX(bconfig)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(duration) * time.Second):
			return
		}
	}
}

// TestIPFIXRun runs a test to verify IPFIX barrage functionality.
func TestIPFIXRun(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	bConfig := models.Config{
		Server:   "127.0.0.1",
		DstPort:  19999,
		SrcRange: "10.10.10.0/28",
		DstRange: "10.11.11.0/28",
		Delay:    1000,
		Workers:  2,
		Web:      false,
		WebIP:    "",
		WebPort:  0,
	}
	testDuration := 30
	sleep := time.Duration(testDuration+5) * time.Second
	// Start receiver
	wg.Add(1)
	go ipfixReceiver(ctx, wg, bConfig.Server, bConfig.DstPort, t)
	// Start IPFIX barrage and run for duration
	wg.Add(1)
	go ipfixRunWrapper(ctx, wg, testDuration, &bConfig)
	// Sleep for longer than expected duration
	time.Sleep(sleep)
	cancel()
	wg.Wait()
}
