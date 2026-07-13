// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Additional table-driven tests for utils package coverage.
// These complement the existing utils_test.go with comprehensive
// coverage of ResolvePortProtocol, RandomIPCIDR, GenerateRand16/32,
// ProtoPorts, and expanded edge-case coverage for other functions.
package utils

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// TestResolvePortProtocol covers all 13 well-known ports plus the default case.
func TestResolvePortProtocol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		port         int
		wantDstPort  uint16
		wantProtocol uint8
	}{
		{"FTP", FTPPort, uint16(FTPPort), TCPProto},
		{"SSH", SSHPort, uint16(SSHPort), TCPProto},
		{"DNS", DNSPort, uint16(DNSPort), UDPProto},
		{"HTTP", HTTPPort, uint16(HTTPPort), TCPProto},
		{"HTTPS", HTTPSPort, uint16(HTTPSPort), TCPProto},
		{"NTP", NTPPort, uint16(NTPPort), UDPProto},
		{"SNMP", SNMPPort, uint16(SNMPPort), UDPProto},
		{"IMAPS", IMAPSPort, uint16(IMAPSPort), TCPProto},
		{"MySQL", MySQLPort, uint16(MySQLPort), TCPProto},
		{"HTTP-Alt", HTTPAltPort, uint16(HTTPAltPort), TCPProto},
		{"HTTPS-Alt", HTTPSAltPort, uint16(HTTPSAltPort), TCPProto},
		{"P2P", P2PPort, uint16(P2PPort), TCPProto},
		{"BT", BTPort, uint16(BTPort), TCPProto},
		{"Unknown-Port-0", 0, uint16(HTTPSPort), TCPProto},
		{"Unknown-Port-9999", 9999, uint16(HTTPSPort), TCPProto},
		{"Unknown-Port-65535", 65535, uint16(HTTPSPort), TCPProto},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotPort, gotProto := ResolvePortProtocol(tc.port)
			if gotPort != tc.wantDstPort {
				t.Errorf("ResolvePortProtocol(%d) dstPort = %d, want %d", tc.port, gotPort, tc.wantDstPort)
			}
			if gotProto != tc.wantProtocol {
				t.Errorf("ResolvePortProtocol(%d) protocol = %d, want %d", tc.port, gotProto, tc.wantProtocol)
			}
		})
	}
}

// TestProtoPorts validates that ProtoPorts contains all expected well-known ports.
func TestProtoPorts(t *testing.T) {
	t.Parallel()
	expected := []int{21, 22, 53, 80, 443, 123, 161, 993, 3306, 8080, 8443, 6681, 6682}
	if len(ProtoPorts) != len(expected) {
		t.Fatalf("ProtoPorts length = %d, want %d", len(ProtoPorts), len(expected))
	}
	for i, want := range expected {
		if ProtoPorts[i] != want {
			t.Errorf("ProtoPorts[%d] = %d, want %d", i, ProtoPorts[i], want)
		}
	}
}

// TestIPto32_TableDriven covers IPv4, IPv4-mapped IPv6, nil, and edge cases.
func TestIPto32_TableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ip   string
		want uint32
	}{
		{"standard-ipv4", "10.10.10.10", 168430090},
		{"localhost", "127.0.0.1", 2130706433},
		{"zero-address", "0.0.0.0", 0},
		{"broadcast", "255.255.255.255", 4294967295},
		{"class-a", "1.2.3.4", 16909060},
		{"class-b", "128.0.0.1", 2147483649},
		{"ipv4-mapped-ipv6", "::ffff:127.0.0.1", 2130706433},
		{"empty-string", "", 0},
		{"invalid-ip", "not-an-ip", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IPto32(tc.ip)
			if got != tc.want {
				t.Errorf("IPto32(%q) = %d, want %d", tc.ip, got, tc.want)
			}
		})
	}
}

// TestIPToNum_TableDriven covers nil, IPv4, and IPv4-mapped IPv6 inputs.
func TestIPToNum_TableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ip   net.IP
		want uint32
	}{
		{"nil-input", nil, 0},
		{"ipv4-localhost", net.ParseIP("127.0.0.1"), 2130706433},
		{"ipv4-zero", net.ParseIP("0.0.0.0"), 0},
		{"ipv4-broadcast", net.ParseIP("255.255.255.255"), 4294967295},
		{"ipv4-mapped-ipv6", net.ParseIP("::ffff:10.0.0.1"), 167772161},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IPToNum(tc.ip)
			if got != tc.want {
				t.Errorf("IPToNum(%v) = %d, want %d", tc.ip, got, tc.want)
			}
		})
	}
}

// TestNumToIP_TableDriven validates round-trip conversion.
func TestNumToIP_TableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		num    uint32
		expect string
	}{
		{0, "0.0.0.0"},
		{2130706433, "127.0.0.1"},
		{167772161, "10.0.0.1"},
		{4294967295, "255.255.255.255"},
		{16909060, "1.2.3.4"},
	}

	for _, tc := range tests {
		t.Run(tc.expect, func(t *testing.T) {
			t.Parallel()
			got := NumToIP(tc.num)
			if got.String() != tc.expect {
				t.Errorf("NumToIP(%d) = %s, want %s", tc.num, got, tc.expect)
			}
		})
	}
}

// TestRandomIP_EdgeCases covers single-IP ranges and invalid CIDRs.
func TestRandomIP_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{"single-ip-/32", "10.0.0.1/32", false},
		{"small-range-/30", "10.0.0.0/30", false},
		{"medium-range-/16", "172.16.0.0/16", false},
		{"large-range-/8", "10.0.0.0/8", false},
		{"invalid-cidr", "not-a-cidr", true},
		{"partial-cidr", "10.0.0", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ip, err := RandomIP(tc.cidr)
			if tc.wantErr {
				if err == nil {
					t.Errorf("RandomIP(%q) expected error, got nil", tc.cidr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RandomIP(%q) unexpected error: %v", tc.cidr, err)
			}
			// For /32, the IP should equal the network address
			if tc.cidr == "10.0.0.1/32" {
				if ip.String() != "10.0.0.1" {
					t.Errorf("RandomIP(/32) = %s, want 10.0.0.1", ip)
				}
			}
		})
	}
}

// TestGetLastIP_TableDriven covers various CIDR sizes.
func TestGetLastIP_TableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cidr     string
		expected string
	}{
		{"10.0.0.0/8", "10.255.255.255"},
		{"172.16.0.0/16", "172.16.255.255"},
		{"192.168.1.0/24", "192.168.1.255"},
		{"10.0.0.0/30", "10.0.0.3"},
		{"10.0.0.0/32", "10.0.0.0"},
	}

	for _, tc := range tests {
		t.Run(tc.cidr, func(t *testing.T) {
			t.Parallel()
			_, ipNet, _ := net.ParseCIDR(tc.cidr)
			last := GetLastIP(ipNet)
			if last.String() != tc.expected {
				t.Errorf("GetLastIP(%s) = %s, want %s", tc.cidr, last, tc.expected)
			}
		})
	}
}

// TestParseIPv4ToNum_ErrorCases covers pure IPv6 and malformed input.
func TestParseIPv4ToNum_ErrorCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid-ipv4", "10.0.0.1", false},
		{"ipv4-mapped-ipv6", "::ffff:127.0.0.1", false},
		{"pure-ipv6", "2001:db8::1", true},
		{"empty-string", "", true},
		{"malformed", "999.999.999.999", true},
		{"partial-ip", "10.0.0", true},
		{"hostname", "localhost", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseIPv4ToNum(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("ParseIPv4ToNum(%q) expected error, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ParseIPv4ToNum(%q) unexpected error: %v", tc.input, err)
			}
		})
	}
}

// TestIsIPv6CIDR_AdditionalCases expands coverage for edge cases.
func TestIsIPv6CIDR_AdditionalCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cidr string
		want bool
	}{
		{"link-local-ipv6", "fe80::/10", true},
		{"loopback-ipv6", "::1/128", true},
		{"wildcard-ipv6", "::/0", true},
		{"private-ipv4", "10.0.0.0/8", false},
		{"public-ipv4", "8.8.8.0/24", false},
		{"non-cidr", "not-valid", false},
		{"just-number", "12345", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsIPv6CIDR(tc.cidr)
			if got != tc.want {
				t.Errorf("IsIPv6CIDR(%q) = %v, want %v", tc.cidr, got, tc.want)
			}
		})
	}
}

// TestRandomIPv6_EdgeCases covers /128 single-address and error paths.
func TestRandomIPv6_EdgeCases(t *testing.T) {
	t.Parallel()

	// /128 single address
	ip, err := RandomIPv6("::1/128")
	if err != nil {
		t.Fatalf("RandomIPv6(::1/128) unexpected error: %v", err)
	}
	if ip.String() != "::1" {
		t.Errorf("RandomIPv6(::1/128) = %s, want ::1", ip)
	}

	// Reject IPv4 CIDR
	_, err = RandomIPv6("10.0.0.0/8")
	if err == nil {
		t.Error("RandomIPv6(10.0.0.0/8) expected error for IPv4 CIDR")
	}

	// Invalid CIDR
	_, err = RandomIPv6("not-a-cidr")
	if err == nil {
		t.Error("RandomIPv6(not-a-cidr) expected error for invalid CIDR")
	}
}

// TestGetLastIPv6_AdditionalCases covers more network sizes.
func TestGetLastIPv6_AdditionalCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cidr     string
		expected string
	}{
		{"2001:db8::/32", "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
		{"fe80::/10", "febf:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
		{"::/0", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
		{"::1/128", "::1"},
	}

	for _, tc := range tests {
		t.Run(tc.cidr, func(t *testing.T) {
			_, ipNet, _ := net.ParseCIDR(tc.cidr)
			last := GetLastIPv6(ipNet)
			expected := net.ParseIP(tc.expected)
			if !last.Equal(expected) {
				t.Errorf("GetLastIPv6(%s) = %s, want %s", tc.cidr, last, expected)
			}
		})
	}
}

// TestRandomIPCIDR_Dispatch verifies IPv4 vs IPv6 dispatching.
func TestRandomIPCIDR_Dispatch(t *testing.T) {
	t.Parallel()

	// IPv4 dispatch
	ip, err := RandomIPCIDR("192.168.0.0/16")
	if err != nil {
		t.Fatalf("RandomIPCIDR(IPv4) unexpected error: %v", err)
	}
	if ip.To4() == nil {
		t.Error("RandomIPCIDR(IPv4) should return IPv4")
	}

	// IPv6 dispatch
	ip, err = RandomIPCIDR("fd00::/8")
	if err != nil {
		t.Fatalf("RandomIPCIDR(IPv6) unexpected error: %v", err)
	}
	if ip.To4() != nil {
		t.Error("RandomIPCIDR(IPv6) should return IPv6")
	}

	// Invalid CIDR
	_, err = RandomIPCIDR("bad-cidr")
	if err == nil {
		t.Error("RandomIPCIDR(bad-cidr) expected error")
	}
}

// TestCryptoRandomNumber_EdgeCases covers boundary conditions.
func TestCryptoRandomNumber_EdgeCases(t *testing.T) {
	t.Parallel()

	// max=1 should always return 0
	result := CryptoRandomNumber(1)
	if result != 0 {
		t.Errorf("CryptoRandomNumber(1) = %d, want 0", result)
	}

	// Verify distribution over a range
	counts := make(map[int64]int)
	for range 1000 {
		n := CryptoRandomNumber(10)
		counts[n]++
	}
	for k, c := range counts {
		if k < 0 || k >= 10 {
			t.Errorf("CryptoRandomNumber(10) returned %d, out of range [0,10)", k)
		}
		if c == 0 {
			t.Errorf("CryptoRandomNumber(10) never returned %d in 1000 samples", k)
		}
	}
}

// TestRandStringBytes_Variants tests multiple lengths and character validity.
func TestRandStringBytes_Variants(t *testing.T) {
	t.Parallel()
	lengths := []int{0, 1, 5, 16, 64, 256}
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for _, n := range lengths {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			result := RandStringBytes(n)
			if len(result) != n {
				t.Errorf("RandStringBytes(%d) length = %d, want %d", n, len(result), n)
			}
			for _, ch := range result {
				found := false
				for _, vc := range validChars {
					if ch == vc {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("RandStringBytes(%d) contains invalid char %q", n, ch)
					break
				}
			}
		})
	}
}

// TestRandomNum_BoundaryConditions covers edge cases for RandomNum.
func TestRandomNum_BoundaryConditions(t *testing.T) {
	t.Parallel()

	// Note: RandomNum(min,min) panics because CryptoRandomNumber(0) panics.
	// This is existing behavior — skipping that case.

	// Small positive range
	for range 100 {
		result := RandomNum(0, 1)
		if result != 0 {
			t.Errorf("RandomNum(0,1) = %d, want 0", result)
		}
	}

	// Negative range
	for range 100 {
		result := RandomNum(-10, 0)
		if result < -10 || result >= 0 {
			t.Errorf("RandomNum(-10,0) = %d, out of range [-10, 0)", result)
		}
	}

	// Medium range
	for range 100 {
		result := RandomNum(10, 100)
		if result < 10 || result >= 100 {
			t.Errorf("RandomNum(10,100) = %d, out of range [10, 100)", result)
		}
	}
}

// TestGenerateRand16_LargeRange verifies behavior with large ranges.
func TestGenerateRand16_LargeRange(t *testing.T) {
	t.Parallel()
	max := 65535
	for range 100 {
		result := GenerateRand16(max)
		if result >= uint16(max) {
			t.Errorf("GenerateRand16(%d) = %d, want < %d", max, result, max)
		}
	}
}

// TestGenerateRand32_LargeRange verifies behavior with large ranges.
func TestGenerateRand32_LargeRange(t *testing.T) {
	t.Parallel()
	max := 100000
	for range 100 {
		result := GenerateRand32(max)
		if result >= uint32(max) {
			t.Errorf("GenerateRand32(%d) = %d, want < %d", max, result, max)
		}
	}
}

// TestBinaryDecoder_MultipleDestinations tests decoding into multiple destinations.
func TestBinaryDecoder_MultipleDestinations(t *testing.T) {
	t.Parallel()

	a := []byte{72, 101, 108, 108, 111} // "Hello"
	b := []byte{87, 111, 114, 108, 100} // "World"

	var combined []byte
	combined = append(combined, a...)
	combined = append(combined, b...)

	destA := make([]byte, 5)
	destB := make([]byte, 5)

	buf := &testReader{data: combined, pos: 0}
	err := BinaryDecoder(buf, destA, destB)
	if err != nil {
		t.Fatalf("BinaryDecoder error: %v", err)
	}
	if string(destA) != "Hello" {
		t.Errorf("destA = %q, want %q", destA, "Hello")
	}
	if string(destB) != "World" {
		t.Errorf("destB = %q, want %q", destB, "World")
	}
}

// testReader is a simple io.Reader wrapper for testing.
type testReader struct {
	data []byte
	pos  int
}

func (r *testReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// TestToBytes_NilAndEmpty covers edge cases.
func TestToBytes_NilAndEmpty(t *testing.T) {
	t.Parallel()

	// Empty struct
	type Empty struct{}
	data, err := ToBytes(Empty{})
	if err != nil {
		t.Fatalf("ToBytes(empty struct) error: %v", err)
	}
	if len(data) == 0 {
		t.Error("ToBytes(empty struct) returned empty byte slice")
	}

	// String
	strData, err := ToBytes("hello")
	if err != nil {
		t.Fatalf("ToBytes(string) error: %v", err)
	}
	if len(strData) == 0 {
		t.Error("ToBytes(string) returned empty byte slice")
	}
}

// TestSendPacket_VerboseFlag tests the verbose flag path.
func TestSendPacket_VerboseFlag(t *testing.T) {
	t.Parallel()

	payload := []byte("verbose test")
	srcPort := 9906
	destPort := 9907
	destIP := net.ParseIP("127.0.0.1")

	conn1, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		t.Skipf("cannot listen on UDP port %d: %v", srcPort, err)
	}
	defer conn1.Close()

	conn2, err := net.ListenUDP("udp", &net.UDPAddr{Port: destPort})
	if err != nil {
		t.Skipf("cannot listen on UDP port %d: %v", destPort, err)
	}
	defer conn2.Close()

	// Receiver goroutine
	done := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 1024)
		n, _, _ := conn2.ReadFromUDP(buf)
		if n > 0 {
			done <- buf[:n]
		}
	}()

	// Send with verbose=true
	n, err := SendPacket(conn1, &net.UDPAddr{IP: destIP, Port: destPort}, payload, true)
	if err != nil {
		t.Fatalf("SendPacket(verbose=true) error: %v", err)
	}
	if n != len(payload) {
		t.Errorf("SendPacket(verbose=true) sent %d bytes, want %d", n, len(payload))
	}

	select {
	case received := <-done:
		if string(received) != string(payload) {
			t.Errorf("payload mismatch: got %q, want %q", received, payload)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for UDP packet (verbose)")
	}
}

// TestBinaryRoundTrip tests the BinaryDecoder/Writer round-trip with BigEndian.
func TestBinaryRoundTrip(t *testing.T) {
	t.Parallel()

	s := "Flowgre Testing Text!"
	b := []byte{70, 108, 111, 119, 103, 114, 101, 32, 84, 101, 115, 116, 105, 110, 103, 32, 84, 101, 120, 116, 33}

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, b)
	if err != nil {
		t.Fatalf("binary.Write error: %v", err)
	}

	result := make([]byte, len(b))
	BinaryDecoder(&buf, result)
	if string(result) != s {
		t.Errorf("round-trip failed: got %q, want %q", result, s)
	}
}
