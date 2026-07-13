//go:build bench

// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package bench contains performance benchmarks for flowgre.
// Run with: go test -tags=bench -bench=. -benchmem ./bench/
package bench

import (
	"net"
	"testing"

	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/netflow"
)

// --- Session Benchmarks ---

// BenchmarkSession_NewSession measures the cost of creating a new Session.
func BenchmarkSession_NewSession(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = netflow.NewSession()
	}
}

// BenchmarkSession_NextSeq measures the cost of incrementing the flow sequence counter.
func BenchmarkSession_NextSeq(b *testing.B) {
	session := netflow.NewSession()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = session.NextSeq()
	}
}

// --- NetFlow Serialization Benchmarks ---

// BenchmarkNetflow_FullPacket_Generic measures the full generation pipeline for a
// NetFlow v9 packet with template + data (generic 18-field profile).
func BenchmarkNetflow_FullPacket_Generic(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 10
	sourceID := 100
	session := netflow.NewSession()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		nf, err := netflow.GenerateNetflow(flowCount, sourceID, srcRange, dstRange, session)
		if err != nil {
			b.Fatal(err)
		}
		_ = nf.ToBytes()
	}
}

// BenchmarkNetflow_DataOnly_Generic measures the hot path: data-only NetFlow v9
// generation + serialization (no template, what barrage sends every cycle).
func BenchmarkNetflow_DataOnly_Generic(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		nf, err := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session)
		if err != nil {
			b.Fatal(err)
		}
		_ = nf.ToBytes()
	}
}

// BenchmarkNetflow_DataOnly_Minimal measures data-only generation with the minimal (7-field) profile.
func BenchmarkNetflow_DataOnly_Minimal(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		nf, err := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session, &netflow.MinimalProfile{})
		if err != nil {
			b.Fatal(err)
		}
		_ = nf.ToBytes()
	}
}

// BenchmarkNetflow_DataOnly_Extended measures data-only generation with the extended (15-field) profile.
func BenchmarkNetflow_DataOnly_Extended(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		nf, err := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session, &netflow.ExtendedProfile{})
		if err != nil {
			b.Fatal(err)
		}
		_ = nf.ToBytes()
	}
}

// --- IPFIX Serialization Benchmarks ---

// BenchmarkIPFIX_DataOnly_Generic measures data-only IPFIX generation + serialization.
func BenchmarkIPFIX_DataOnly_Generic(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		pkt, err := ipfix.GenerateDataIPFIX(flowCount, sourceID, srcRange, dstRange, 0, session)
		if err != nil {
			b.Fatal(err)
		}
		_ = pkt.ToBytes()
	}
}

// --- Timestamp Update Benchmarks ---

// BenchmarkNetflow_UpdateTimeStamp measures the cost of updating the timestamp
// in an existing NetFlow v9 packet (replay path).
func BenchmarkNetflow_UpdateTimeStamp(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()
	nf, err := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session)
	if err != nil {
		b.Fatal(err)
	}
	buf := nf.ToBytes()
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = netflow.UpdateTimeStamp(data)
	}
}

// BenchmarkIPFIX_UpdateTimeStamp measures the cost of updating the timestamp
// in an existing IPFIX packet (replay path).
func BenchmarkIPFIX_UpdateTimeStamp(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()
	pkt, err := ipfix.GenerateDataIPFIX(flowCount, sourceID, srcRange, dstRange, 0, session)
	if err != nil {
		b.Fatal(err)
	}
	buf := pkt.ToBytes()
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = ipfix.UpdateTimeStamp(data)
	}
}

// --- Validation Benchmarks ---

// BenchmarkValidate_NetFlow_Valid measures parsing a valid NetFlow v9 header.
func BenchmarkValidate_NetFlow_Valid(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()
	nf, err := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session)
	if err != nil {
		b.Fatal(err)
	}
	buf := nf.ToBytes()
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = netflow.IsValidNetFlow(data, 9)
	}
}

// BenchmarkValidate_NetFlow_Invalid measures validating with wrong version.
func BenchmarkValidate_NetFlow_Invalid(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()
	nf, err := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session)
	if err != nil {
		b.Fatal(err)
	}
	buf := nf.ToBytes()
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = netflow.IsValidNetFlow(data, 10) // wrong version
	}
}

// BenchmarkValidate_IPFIX_Valid measures parsing a valid IPFIX header.
func BenchmarkValidate_IPFIX_Valid(b *testing.B) {
	srcRange := "10.0.0.0/8"
	dstRange := "172.16.0.0/12"
	flowCount := 15
	sourceID := 100
	session := netflow.NewSession()
	pkt, err := ipfix.GenerateDataIPFIX(flowCount, sourceID, srcRange, dstRange, 0, session)
	if err != nil {
		b.Fatal(err)
	}
	buf := pkt.ToBytes()
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = netflow.IsValidNetFlow(data, 10)
	}
}

// --- UDP Send Benchmarks ---

// BenchmarkUDPSend_Small measures sending a 100-byte UDP datagram.
func BenchmarkUDPSend_Small(b *testing.B) {
	sender, listener := newLocalpair(b)
	data := make([]byte, 100)
	addr := listener.LocalAddr().(*net.UDPAddr)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = sender.WriteTo(data, addr)
	}
}

// BenchmarkUDPSend_Medium measures sending a 764-byte UDP datagram (typical flow packet).
func BenchmarkUDPSend_Medium(b *testing.B) {
	sender, listener := newLocalpair(b)
	data := make([]byte, 764)
	addr := listener.LocalAddr().(*net.UDPAddr)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = sender.WriteTo(data, addr)
	}
}

// BenchmarkUDPSend_Large measures sending a 2048-byte UDP datagram.
func BenchmarkUDPSend_Large(b *testing.B) {
	sender, listener := newLocalpair(b)
	data := make([]byte, 2048)
	addr := listener.LocalAddr().(*net.UDPAddr)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = sender.WriteTo(data, addr)
	}
}

// newLocalpair creates a sender + receiver UDP socket pair on localhost.
func newLocalpair(b *testing.B) (*net.UDPConn, *net.UDPConn) {
	b.Helper()

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		b.Fatalf("failed to create listener: %v", err)
	}
	b.Cleanup(func() { listener.Close() })

	sender, err := net.DialUDP("udp", nil, listener.LocalAddr().(*net.UDPAddr))
	if err != nil {
		b.Fatalf("failed to create sender: %v", err)
	}
	b.Cleanup(func() { sender.Close() })

	return sender, listener
}
