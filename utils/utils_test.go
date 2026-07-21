// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Util funcs used throughout Flowgre

package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"net"
	"strings"
	"testing"
	"time"
)

// TestRandStringBytes
func TestRandStringBytes(t *testing.T) {
	t.Parallel()
	n := 16
	result, err := RandStringBytes(n)
	if err != nil {
		t.Fatalf("RandStringBytes(%d) error: %v", n, err)
	}
	if len(result) != n {
		t.Errorf("Result was improper length. Got: %d Want: %d", len(result), n)
	}
	if strings.Contains(result, "NoneRandom") {
		t.Errorf("Result was NOT random!")
	}
}

// TestGenerateRand16
func TestGenerateRand16(t *testing.T) {
	t.Parallel()
	n := 16
	result, err := GenerateRand16(n)
	if err != nil {
		t.Fatalf("GenerateRand16(%d) error: %v", n, err)
	}
	if result > uint16(n) {
		t.Errorf("Result was larger than expected max! Got: %d Want less than: %d", result, n)
	}
}

func TestGenerateRand32(t *testing.T) {
	t.Parallel()
	n := 16
	result, err := GenerateRand32(n)
	if err != nil {
		t.Fatalf("GenerateRand32(%d) error: %v", n, err)
	}
	if result > uint32(n) {
		t.Errorf("Result was larger than expected max! Got: %d Want less than: %d", result, n)
	}
}

func TestBinaryDecoder(t *testing.T) {
	t.Parallel()
	s := "Flowgre Testing Text!"
	var buf bytes.Buffer
	// This is the byte version of s above.  Doing explicitly for testing sake.
	b := []byte{70, 108, 111, 119, 103, 114, 101, 32, 84, 101, 115, 116, 105, 110, 103, 32, 84, 101, 120, 116, 33}
	err := binary.Write(&buf, binary.BigEndian, b)
	if err != nil {
		t.Errorf("Unable to write binary data to buffer: %s", err)
	}
	// var result []byte
	result := make([]byte, len(b))
	BinaryDecoder(&buf, result)
	if string(result) != s {
		t.Errorf("Result was not the expected string! Got: %s Want: %s", result, s)
	}
}

func TestIPto32(t *testing.T) {
	t.Parallel()
	// "10.10.10.10"
	bip := uint32(168430090)
	result := IPto32("10.10.10.10")
	if result != bip {
		t.Errorf("Result didn't match! Got: %d Want: %d", result, bip)
	}
}

func TestRandomNum(t *testing.T) {
	t.Parallel()
	count := 10000
	for range count {
		min := 10
		max := 250
		result, err := RandomNum(min, max)
		if err != nil {
			t.Fatalf("RandomNum(%d, %d) error: %v", min, max, err)
		}
		if result > max {
			t.Errorf("Result is greater than max! Got: %d Want: %d", result, max)
		}
		if result < min {
			t.Errorf("Result is less than min! Got: %d Want: %d", result, min)
		}
		if result == 0 {
			t.Errorf("Result is less than min! Got: %d Want: %d", result, min)
		}
	}
	t.Logf("Successfully generated %d random ints", count)
}

func TestToBytes(t *testing.T) {
	t.Parallel()

	// Test with a simple struct
	type TestStruct struct {
		Name  string
		Value int
	}
	original := TestStruct{Name: "flowgre", Value: 42}

	data, err := ToBytes(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty byte slice")
	}

	// Verify we can decode it back
	var buf bytes.Buffer
	buf.Write(data)
	dec := gob.NewDecoder(&buf)
	var decoded TestStruct
	if err := dec.Decode(&decoded); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if decoded.Name != original.Name {
		t.Errorf("expected Name %q, got %q", original.Name, decoded.Name)
	}
	if decoded.Value != original.Value {
		t.Errorf("expected Value %d, got %d", original.Value, decoded.Value)
	}
}

func TestRandomIP(t *testing.T) {
	t.Parallel()
	const (
		cidr = "10.0.0.0/8"
		itr  = 10000
	)
	for range itr {
		_, ipNet, _ := net.ParseCIDR(cidr)
		result, _ := RandomIP(cidr)

		if !ipNet.Contains(result) {
			t.Errorf("Result isn't within bounds! Got: %s Want: %s", result, cidr)
		}
		//else {
		//	t.Logf("Result %s is found in %s", result, cidr)
		//}
	}
	t.Logf("Generated %d random IPs inside %s", itr, cidr)
}

// TestParseIPv4ToNum tests the ParseIPv4ToNum function for correct IPv4 parsing,
// pure IPv6 rejection, and invalid input handling.
func TestParseIPv4ToNum(t *testing.T) {
	t.Parallel()

	num, err := ParseIPv4ToNum("10.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error for valid IPv4: %v", err)
	}
	if num != 167772161 {
		t.Errorf("unexpected value: got %d, want %d", num, 167772161)
	}

	_, err = ParseIPv4ToNum("::1")
	if err == nil {
		t.Error("expected error for pure IPv6")
	}

	_, err = ParseIPv4ToNum("not-an-ip")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestSendPacket(t *testing.T) {
	t.Parallel()
	payload1 := []byte("Flowgre Testing Text!")
	payload2 := make([]byte, len(payload1))
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, payload1)
	if err != nil {
		t.Errorf("Unable to write binary data to buffer: %s", err)
	}
	srcPort := 9905
	destPort := 9995
	destIP := net.ParseIP("127.0.0.1")
	conn1, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		t.Errorf("Listening on UDP port 9905 failed! Got: %s", err)
	}
	conn2, err := net.ListenUDP("udp", &net.UDPAddr{Port: destPort})
	if err != nil {
		t.Errorf("Listening on UDP port 9995 failed! Got: %s", err)
	}

	SendPacket(conn1, &net.UDPAddr{IP: destIP, Port: destPort}, buf.Bytes(), false)

	_, _, err = conn2.ReadFromUDP(payload2)
	if err != nil {
		t.Errorf("Failed to reieve UDP packet! Got: %s", err)
	}
	if string(payload1) != string(payload2) {
		t.Errorf("Failed to get proper packet!")
	}
}

func TestIsIPv6CIDR(t *testing.T) {
	t.Parallel()

	// IPv6 CIDRs
	if !IsIPv6CIDR("2001:db8::/32") {
		t.Error("expected 2001:db8::/32 to be IPv6")
	}
	if !IsIPv6CIDR("fe80::/10") {
		t.Error("expected fe80::/10 to be IPv6")
	}
	if !IsIPv6CIDR("::1/128") {
		t.Error("expected ::1/128 to be IPv6")
	}

	// IPv4 CIDRs
	if IsIPv6CIDR("10.0.0.0/8") {
		t.Error("expected 10.0.0.0/8 to NOT be IPv6")
	}
	if IsIPv6CIDR("192.168.1.0/24") {
		t.Error("expected 192.168.1.0/24 to NOT be IPv6")
	}

	// Invalid
	if IsIPv6CIDR("not-a-cidr") {
		t.Error("expected invalid CIDR to return false")
	}
}

func TestRandomIPv6(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cidr     string
		expected net.IP // prefix that must match
	}{
		{"2001:db8::/32", net.ParseIP("2001:db8::")},
		{"fe80::/10", net.ParseIP("fe80::")},
		{"2001:db8:1::/48", net.ParseIP("2001:db8:1::")},
		{"2001:db8:1:2::/64", net.ParseIP("2001:db8:1:2::")},
		{"::1/128", net.ParseIP("::1")},
		{"::/0", net.ParseIP("::")},
	}

	for _, tc := range testCases {
		t.Run(tc.cidr, func(t *testing.T) {
			_, ipNet, _ := net.ParseCIDR(tc.cidr)
			for range 1000 {
				result, err := RandomIPv6(tc.cidr)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !ipNet.Contains(result) {
					t.Errorf("IP %s not in CIDR %s", result, tc.cidr)
				}
				if result.To4() != nil {
					t.Errorf("expected IPv6, got IPv4: %s", result)
				}
			}
		})
	}
}

func TestRandomIPv6RejectsIPv4(t *testing.T) {
	t.Parallel()

	_, err := RandomIPv6("10.0.0.0/8")
	if err == nil {
		t.Error("expected error for IPv4 CIDR")
	}
}

func TestGetLastIPv6(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cidr     string
		expected string
	}{
		{"2001:db8::/32", "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
		{"fe80::/10", "febf:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
		{"2001:db8:1::/48", "2001:db8:1:ffff:ffff:ffff:ffff:ffff"},
		{"2001:db8:1:2::/64", "2001:db8:1:2:ffff:ffff:ffff:ffff"},
		{"::1/128", "::1"},
		{"::/0", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
	}

	for _, tc := range testCases {
		t.Run(tc.cidr, func(t *testing.T) {
			_, ipNet, _ := net.ParseCIDR(tc.cidr)
			last := GetLastIPv6(ipNet)
			expected := net.ParseIP(tc.expected)
			if !last.Equal(expected) {
				t.Errorf("expected %s, got %s", expected, last)
			}
		})
	}
}

func TestRandomIPCIDR(t *testing.T) {
	t.Parallel()

	// IPv4 CIDR should return IPv4
	ip, err := RandomIPCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip.To4() == nil {
		t.Error("expected IPv4 for IPv4 CIDR")
	}

	// IPv6 CIDR should return IPv6
	ip, err = RandomIPCIDR("2001:db8::/32")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip.To4() != nil {
		t.Error("expected IPv6 for IPv6 CIDR")
	}
}

func TestSendPacket_IPv6(t *testing.T) {
	t.Parallel()

	// Start listener on IPv6
	listener, err := net.ListenUDP("udp6", &net.UDPAddr{IP: net.ParseIP("::1"), Port: 0})
	if err != nil {
		t.Skipf("IPv6 not available: %v", err)
	}
	defer listener.Close()

	// Start receiver goroutine
	done := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 1024)
		n, _, _ := listener.ReadFromUDP(buf)
		if n > 0 {
			done <- buf[:n]
		}
	}()

	// Send packet to IPv6 address
	sender, err := net.ListenUDP("udp6", &net.UDPAddr{Port: 0})
	if err != nil {
		t.Fatalf("failed to create sender: %v", err)
	}
	defer sender.Close()

	listenerAddr := listener.LocalAddr().(*net.UDPAddr)
	payload := []byte("IPv6 test payload")
	_, err = SendPacket(sender, &net.UDPAddr{IP: net.ParseIP("::1"), Port: listenerAddr.Port}, payload, false)
	if err != nil {
		t.Fatalf("SendPacket failed: %v", err)
	}

	select {
	case received := <-done:
		if string(received) != string(payload) {
			t.Errorf("payload mismatch: got %q, want %q", received, payload)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for IPv6 packet")
	}
}
