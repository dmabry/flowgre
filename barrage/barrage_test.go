// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package barrage

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/dmabry/flowgre/flow/netflow"
	"github.com/dmabry/flowgre/models"
	"net"
	"sync"
	"testing"
	"time"
)

const udpMaxBufferSize = 65507

func runWrapper(ctx context.Context, wg *sync.WaitGroup, duration int, bconfig *models.Config) {
	defer wg.Done()
	go Run(bconfig)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(duration) * time.Second):
			return
		}
	}
}

func receiver(ctx context.Context, wg *sync.WaitGroup, ip string, port int, t *testing.T) {
	defer wg.Done()
	parsedFlows := 0

	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		t.Fatalf("Test Receiver listening on %s:%d failed! Got: %v", ip, port, err)
	}
	t.Logf("Test Receiver listening on %s:%d", ip, port)
	defer func(conn *net.UDPConn) {
		err := conn.Close()
		if err != nil {
			t.Fatalf("Error closing listener: %v", err)
		}
	}(conn)
	// Start the loop and check context for done, otherwise listen for packets
	for {
		select {
		case <-ctx.Done():
			t.Log("Test Receiver exiting due to signal")
			t.Logf("Parsed flows: %d", parsedFlows)
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
				// No packets received before deadline moving on ...
				continue
			}
			payload = payload[:length]
			ok, err := netflow.IsValidNetFlow(payload, 9)
			if !ok {
				t.Errorf("Invalid NetFlow Packet: %v", err)
				return
			}
			header := netflow.Header{}
			reader := bytes.NewReader(payload)
			var flowSetID uint16
			var flowLength uint16
			// read header
			err = binary.Read(reader, binary.BigEndian, &header)
			if err != nil {
				t.Errorf("Failed to parse Netflow Header! Got: %v", err)
				return
			}
			// read flowSetID
			err = binary.Read(reader, binary.BigEndian, &flowSetID)
			if err != nil {
				t.Errorf("Failed to parse Netflow flowSetID! Got: %v", err)
				return
			}
			// read flowLength
			err = binary.Read(reader, binary.BigEndian, &flowLength)
			if err != nil {
				t.Errorf("Failed to parse Netflow flowLength! Got: %v", err)
				return
			}
			// read all flows from the payload
			count := int(header.FlowCount)
			for i := 0; i < count; i++ {
				flow := netflow.GenericFlow{}
				err := binary.Read(reader, binary.BigEndian, &flow)
				if err != nil {
					t.Errorf("Issue reading in GenericFlow")
					return
				}
				parsedFlows++
			}
		}
	}

}

// TestRun runs a test to verify functionality.
func TestRun(t *testing.T) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	// Create known config
	bConfig := models.Config{
		Server:   "127.0.0.1",
		DstPort:  9995,
		SrcRange: "10.10.10.0/28",
		DstRange: "10.11.11.0/28",
		Delay:    1000,
		Workers:  2,
		Web:      false,
		WebIP:    "",
		WebPort:  0,
	}
	testDuration := 60
	sleep := time.Duration(testDuration+5) * time.Second
	// Start receiver
	wg.Add(1)
	go receiver(ctx, wg, bConfig.Server, bConfig.DstPort, t)
	// Start barrage and run for duration
	wg.Add(1)
	go runWrapper(ctx, wg, testDuration, &bConfig)
	// Sleep for longer than expected duration
	time.Sleep(sleep)
	cancel()
	wg.Wait()
	// verify payload received via listener
}
