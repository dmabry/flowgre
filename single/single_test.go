// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package single

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/dmabry/flowgre/flow/netflow"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

const udpMaxBufferSize = 65507

func receiver(ctx context.Context, wg *sync.WaitGroup, ip string, port int, t *testing.T, errs chan<- error) {
	defer wg.Done()
	parsedFlows := 0

	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		errs <- fmt.Errorf("test receiver listening on %s:%d failed: %w", ip, port, err)
	}
	t.Logf("Test Receiver listening on %s:%d", ip, port)
	defer func(conn *net.UDPConn) {
		err := conn.Close()
		if err != nil {
			errs <- fmt.Errorf("error closing listener: %w", err)
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
				errs <- fmt.Errorf("issue setting deadline: %w", err)
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
				errs <- fmt.Errorf("invalid NetFlow packet: %w", err)
				return
			}
			header := netflow.Header{}
			reader := bytes.NewReader(payload)
			var flowSetID uint16
			var flowLength uint16
			// read header
			err = binary.Read(reader, binary.BigEndian, &header)
			if err != nil {
				errs <- fmt.Errorf("failed to parse NetFlow header: %w", err)
				return
			}
			// read flowSetID
			err = binary.Read(reader, binary.BigEndian, &flowSetID)
			if err != nil {
				errs <- fmt.Errorf("failed to parse NetFlow flowSetID: %w", err)
				return
			}
			// read flowLength
			err = binary.Read(reader, binary.BigEndian, &flowLength)
			if err != nil {
				errs <- fmt.Errorf("failed to parse NetFlow flowLength: %w", err)
				return
			}
			// read all flows from the payload
			count := int(header.FlowCount)
			for i := 0; i < count; i++ {
				flow := netflow.GenericFlow{}
				err := binary.Read(reader, binary.BigEndian, &flow)
				if err != nil {
					errs <- fmt.Errorf("issue reading GenericFlow: %w", err)
					return
				}
				parsedFlows++
			}
		}
	}

}

// TestRun runs a test to verify functionality.
func TestRun(t *testing.T) {
	t.Parallel()
	os.Stdout, _ = os.Open(os.DevNull) // hide all stdout from single
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	sleep := 2 * time.Second
	// Start receiver
	wg.Add(1)
	errs := make(chan error)
go receiver(ctx, wg, "127.0.0.1", 9997, t, errs)
	// Run single
	Run("127.0.0.1", 9997, 9999, 10, "10.10.10.0/28", "10.11.11.0/28", false)
	// Sleep for longer than expected duration
	time.Sleep(sleep)
	cancel()
	wg.Wait()
close(errs)
for err := range errs {
    t.Errorf("Received error: %v", err)
}
}
