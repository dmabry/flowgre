// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Util funcs used throughout Flowgre

package utils

import (
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"testing"
)

// TestRandStringBytes
func TestRandStringBytes(t *testing.T) {
	n := 16
	result := RandStringBytes(n)
	if len(result) != n {
		t.Errorf("Result was improper length. Got: %d Want: %d", len(result), n)
	}
	if strings.Contains(result, "NoneRandom") {
		t.Errorf("Result was NOT random!")
	}
}

// TestGenerateRand16
func TestGenerateRand16(t *testing.T) {
	n := 16
	result := GenerateRand16(n)
	if result > uint16(n) {
		t.Errorf("Result was larger than expected max! Got: %d Want less than: %d", result, n)
	}
}

func TestGenerateRand32(t *testing.T) {
	n := 16
	result := GenerateRand32(n)
	if result > uint32(n) {
		t.Errorf("Result was larger than expected max! Got: %d Want less than: %d", result, n)
	}
}

func TestBinaryDecoder(t *testing.T) {
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
	// "10.10.10.10"
	bip := uint32(168430090)
	result := IPto32("10.10.10.10")
	if result != bip {
		t.Errorf("Result didn't match! Got: %d Want: %d", result, bip)
	}
}

func TestRandomNum(t *testing.T) {
	count := 10000
	for i := 0; i < count; i++ {
		min := 10
		max := 250
		result := RandomNum(min, max)
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
	//Stub TODO: Make a test that is actually applicable.  The function isn't used currently
}

func TestRandomIP(t *testing.T) {
	const (
		cidr = "10.0.0.0/8"
		itr  = 10000
	)
	for i := 0; i < itr; i++ {
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

func TestSendPacket(t *testing.T) {
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

	SendPacket(conn1, &net.UDPAddr{IP: destIP, Port: destPort}, buf, false)

	_, _, err = conn2.ReadFromUDP(payload2)
	if err != nil {
		t.Errorf("Failed to reieve UDP packet! Got: %s", err)
	}
	if string(payload1) != string(payload2) {
		t.Errorf("Failed to get proper packet!")
	}
}
