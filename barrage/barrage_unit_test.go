// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package barrage

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/netflow"
)

// TestNetFlowGenerator produces valid NetFlow v9 packets.
func TestNetFlowGenerator(t *testing.T) {
	t.Parallel()
	gen := NetFlow()
	session := netflow.NewSession()

	if gen.Label() != "Worker" {
		t.Errorf("Label wrong: got %q, want %q", gen.Label(), "Worker")
	}

	tBuf := gen.GenerateTemplate(42, session)
	if len(tBuf) == 0 {
		t.Fatal("GenerateTemplate returned empty buffer")
	}
	ok, err := netflow.IsValidNetFlow(tBuf, 9)
	if err != nil || !ok {
		t.Errorf("GenerateTemplate produced invalid NetFlow: %v", err)
	}

	dBuf := gen.GenerateData(10, 42, "10.0.0.0/8", "10.0.0.0/8", session)
	if len(dBuf) == 0 {
		t.Fatal("GenerateData returned empty buffer")
	}
	ok, err = netflow.IsValidNetFlow(dBuf, 9)
	if err != nil || !ok {
		t.Errorf("GenerateData produced invalid NetFlow: %v", err)
	}
}

// TestIPFIXGenerator produces valid IPFIX v10 packets.
func TestIPFIXGenerator(t *testing.T) {
	t.Parallel()
	gen := IPFIX()
	session := netflow.NewSession()

	if gen.Label() != "IPFIX Worker" {
		t.Errorf("Label wrong: got %q, want %q", gen.Label(), "IPFIX Worker")
	}

	tBuf := gen.GenerateTemplate(42, session)
	if len(tBuf) == 0 {
		t.Fatal("GenerateTemplate returned empty buffer")
	}
	ok, err := ipfix.IsValidIPFIX(tBuf)
	if err != nil || !ok {
		t.Errorf("GenerateTemplate produced invalid IPFIX: %v", err)
	}

	dBuf := gen.GenerateData(10, 42, "10.0.0.0/8", "10.0.0.0/8", session)
	if len(dBuf) == 0 {
		t.Fatal("GenerateData returned empty buffer")
	}
	ok, err = ipfix.IsValidIPFIX(dBuf)
	if err != nil || !ok {
		t.Errorf("GenerateData produced invalid IPFIX: %v", err)
	}
}

// TestRunCtxStopsOnCancel verifies that RunCtx stops all workers
// when the context is cancelled.
func TestRunCtxStopsOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()
	port := listener.LocalAddr().(*net.UDPAddr).Port

	// Drain packets so SendPacket doesn't block
	ready := make(chan struct{})
	go func() {
		close(ready)
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				listener.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				_, _, err := listener.ReadFromUDP(buf)
				if err != nil {
					continue
				}
			}
		}
	}()
	<-ready

	config := &models.Config{
		Server:           "127.0.0.1",
		DstPort:          port,
		SrcRange:         "10.0.0.0/24",
		DstRange:         "10.0.0.0/24",
		Workers:          1,
		Delay:            50,
		TemplateInterval: 30,
	}

	done := make(chan struct{})
	go func() {
		RunCtx(ctx, config, NetFlow())
		close(done)
	}()

	// Give worker time to start
	time.Sleep(200 * time.Millisecond)

	// Cancel and verify clean shutdown
	cancel()
	select {
	case <-done:
		// Clean shutdown
	case <-time.After(3 * time.Second):
		t.Fatal("RunCtx did not exit within 3 seconds after context cancellation")
	}
}

// TestRunCtxSendsValidPackets verifies that workers send valid protocol packets.
func TestRunCtxSendsValidPackets(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()
	port := listener.LocalAddr().(*net.UDPAddr).Port

	var mu sync.Mutex
	var validPackets int

	ready := make(chan struct{})
	go func() {
		close(ready)
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				listener.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, _, err := listener.ReadFromUDP(buf)
				if err != nil {
					continue
				}
				pkt := buf[:n]
				ok, _ := netflow.IsValidNetFlow(pkt, 9)
				if ok {
					mu.Lock()
					validPackets++
					mu.Unlock()
				}
			}
		}
	}()
	<-ready

	config := &models.Config{
		Server:           "127.0.0.1",
		DstPort:          port,
		SrcRange:         "10.0.0.0/24",
		DstRange:         "10.0.0.0/24",
		Workers:          1,
		Delay:            50,
		TemplateInterval: 30,
	}

	RunCtx(ctx, config, NetFlow())

	mu.Lock()
	count := validPackets
	mu.Unlock()

	if count == 0 {
		t.Error("No valid packets received")
	}
	t.Logf("Received %d valid NetFlow packets", count)
}

// TestRunCtxIPFIXSendsValidPackets verifies IPFIX workers send valid packets.
func TestRunCtxIPFIXSendsValidPackets(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()
	port := listener.LocalAddr().(*net.UDPAddr).Port

	var mu sync.Mutex
	var validPackets int

	ready := make(chan struct{})
	go func() {
		close(ready)
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				listener.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, _, err := listener.ReadFromUDP(buf)
				if err != nil {
					continue
				}
				pkt := buf[:n]
				ok, _ := ipfix.IsValidIPFIX(pkt)
				if ok {
					mu.Lock()
					validPackets++
					mu.Unlock()
				}
			}
		}
	}()
	<-ready

	config := &models.Config{
		Server:           "127.0.0.1",
		DstPort:          port,
		SrcRange:         "10.0.0.0/24",
		DstRange:         "10.0.0.0/24",
		Workers:          1,
		Delay:            50,
		TemplateInterval: 30,
	}

	RunCtx(ctx, config, IPFIX())

	mu.Lock()
	count := validPackets
	mu.Unlock()

	if count == 0 {
		t.Error("No valid packets received")
	}
	t.Logf("Received %d valid IPFIX packets", count)
}

// TestRunCtxTemplateRetransmission verifies templates are retransmitted
// at the configured interval using a 2-second interval.
func TestRunCtxTemplateRetransmission(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()
	port := listener.LocalAddr().(*net.UDPAddr).Port

	var mu sync.Mutex
	var templatesReceived int

	ready := make(chan struct{})
	go func() {
		close(ready)
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				listener.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, _, err := listener.ReadFromUDP(buf)
				if err != nil {
					continue
				}
				pkt := buf[:n]
				if len(pkt) < 22 {
					continue
				}
				version := binary.BigEndian.Uint16(pkt[0:2])
				flowSetID := binary.BigEndian.Uint16(pkt[20:22])
				if version == 9 && flowSetID == 0 {
					mu.Lock()
					templatesReceived++
					mu.Unlock()
				}
			}
		}
	}()
	<-ready

	config := &models.Config{
		Server:           "127.0.0.1",
		DstPort:          port,
		SrcRange:         "10.0.0.0/24",
		DstRange:         "10.0.0.0/24",
		Workers:          1,
		Delay:            50,
		TemplateInterval: 2,
	}

	RunCtx(ctx, config, NetFlow())

	mu.Lock()
	tmpl := templatesReceived
	mu.Unlock()

	// Initial template + at least one retransmit (2s interval, 10s timeout)
	if tmpl < 2 {
		t.Errorf("Expected at least 2 templates, got %d", tmpl)
	}
	t.Logf("Received %d templates with 2s interval", tmpl)
}

// TestRunCtxZeroTemplateInterval verifies that template-interval=0
// disables retransmission.
func TestRunCtxZeroTemplateInterval(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()
	port := listener.LocalAddr().(*net.UDPAddr).Port

	var mu sync.Mutex
	var templatesReceived int

	ready := make(chan struct{})
	go func() {
		close(ready)
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				listener.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, _, err := listener.ReadFromUDP(buf)
				if err != nil {
					continue
				}
				pkt := buf[:n]
				if len(pkt) < 22 {
					continue
				}
				version := binary.BigEndian.Uint16(pkt[0:2])
				flowSetID := binary.BigEndian.Uint16(pkt[20:22])
				if version == 9 && flowSetID == 0 {
					mu.Lock()
					templatesReceived++
					mu.Unlock()
				}
			}
		}
	}()
	<-ready

	config := &models.Config{
		Server:           "127.0.0.1",
		DstPort:          port,
		SrcRange:         "10.0.0.0/24",
		DstRange:         "10.0.0.0/24",
		Workers:          1,
		Delay:            50,
		TemplateInterval: 0,
	}

	RunCtx(ctx, config, NetFlow())

	mu.Lock()
	tmpl := templatesReceived
	mu.Unlock()

	// Should see the initial template (1), but no retransmissions
	if tmpl > 1 {
		t.Errorf("Expected at most 1 template with interval=0, got %d", tmpl)
	}
	t.Logf("Templates with interval=0: %d", tmpl)
}

// TestRunCtxSendsValidPackets_IPv6 verifies that workers send valid NetFlow
// packets with IPv6 flow data to an IPv6 listener.
func TestRunCtxSendsValidPackets_IPv6(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	listener, err := net.ListenUDP("udp6", &net.UDPAddr{IP: net.ParseIP("::1"), Port: 0})
	if err != nil {
		t.Skipf("IPv6 not available: %v", err)
	}
	defer listener.Close()
	port := listener.LocalAddr().(*net.UDPAddr).Port

	var mu sync.Mutex
	var validPackets int

	ready := make(chan struct{})
	go func() {
		close(ready)
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				listener.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, _, err := listener.ReadFromUDP(buf)
				if err != nil {
					continue
				}
				pkt := buf[:n]
				ok, _ := netflow.IsValidNetFlow(pkt, 9)
				if ok {
					mu.Lock()
					validPackets++
					mu.Unlock()
				}
			}
		}
	}()
	<-ready

	config := &models.Config{
		Server:           "::1",
		DstPort:          port,
		SrcRange:         "2001:db8:1::/48",
		DstRange:         "2001:db8:2::/48",
		Workers:          1,
		Delay:            50,
		TemplateInterval: 30,
	}

	RunCtx(ctx, config, NetFlow())

	mu.Lock()
	count := validPackets
	mu.Unlock()

	if count == 0 {
		t.Error("No valid packets received")
	}
	t.Logf("Received %d valid NetFlow packets with IPv6 flows", count)
}
