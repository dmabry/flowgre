// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"net"
	"testing"

	"github.com/dmabry/flowgre/utils"
)

func TestMinimalFlow_Generate_AllProtocols(t *testing.T) {
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

	cases := []struct {
		port         int
		wantPort     uint16
		wantProtocol uint8
	}{
		{utils.SSHPort, uint16(utils.SSHPort), utils.TCPProto},
		{utils.FTPPort, uint16(utils.FTPPort), utils.TCPProto},
		{utils.DNSPort, uint16(utils.DNSPort), utils.UDPProto},
		{utils.HTTPPort, uint16(utils.HTTPPort), utils.TCPProto},
		{utils.HTTPSPort, uint16(utils.HTTPSPort), utils.TCPProto},
		{utils.NTPPort, uint16(utils.NTPPort), utils.UDPProto},
		{utils.SNMPPort, uint16(utils.SNMPPort), utils.UDPProto},
		{utils.IMAPSPort, uint16(utils.IMAPSPort), utils.TCPProto},
		{utils.MySQLPort, uint16(utils.MySQLPort), utils.TCPProto},
		{utils.HTTPAltPort, uint16(utils.HTTPAltPort), utils.TCPProto},
		{utils.HTTPSAltPort, uint16(utils.HTTPSAltPort), utils.TCPProto},
		{utils.P2PPort, uint16(utils.P2PPort), utils.TCPProto},
		{utils.BTPort, uint16(utils.BTPort), utils.TCPProto},
		{99999, uint16(utils.HTTPSPort), utils.TCPProto}, // default
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			session := NewSession()
			mf := new(MinimalFlow).Generate(srcIP, dstIP, tc.port, session)
			if mf.DstPort != tc.wantPort {
				t.Errorf("DstPort: got %d, want %d", mf.DstPort, tc.wantPort)
			}
			if mf.Protocol != tc.wantProtocol {
				t.Errorf("Protocol: got %d, want %d", mf.Protocol, tc.wantProtocol)
			}
		})
	}
}

func TestExtendedFlow_Generate_AllProtocols(t *testing.T) {
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

	cases := []struct {
		port         int
		wantPort     uint16
		wantProtocol uint8
	}{
		{utils.SSHPort, uint16(utils.SSHPort), utils.TCPProto},
		{utils.FTPPort, uint16(utils.FTPPort), utils.TCPProto},
		{utils.DNSPort, uint16(utils.DNSPort), utils.UDPProto},
		{utils.HTTPPort, uint16(utils.HTTPPort), utils.TCPProto},
		{utils.HTTPSPort, uint16(utils.HTTPSPort), utils.TCPProto},
		{utils.NTPPort, uint16(utils.NTPPort), utils.UDPProto},
		{utils.SNMPPort, uint16(utils.SNMPPort), utils.UDPProto},
		{utils.IMAPSPort, uint16(utils.IMAPSPort), utils.TCPProto},
		{utils.MySQLPort, uint16(utils.MySQLPort), utils.TCPProto},
		{utils.HTTPAltPort, uint16(utils.HTTPAltPort), utils.TCPProto},
		{utils.HTTPSAltPort, uint16(utils.HTTPSAltPort), utils.TCPProto},
		{utils.P2PPort, uint16(utils.P2PPort), utils.TCPProto},
		{utils.BTPort, uint16(utils.BTPort), utils.TCPProto},
		{99999, uint16(utils.HTTPSPort), utils.TCPProto}, // default
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			session := NewSession()
			ef := new(ExtendedFlow).Generate(srcIP, dstIP, tc.port, session)
			if ef.DstPort != tc.wantPort {
				t.Errorf("DstPort: got %d, want %d", ef.DstPort, tc.wantPort)
			}
			if ef.Protocol != tc.wantProtocol {
				t.Errorf("Protocol: got %d, want %d", ef.Protocol, tc.wantProtocol)
			}
		})
	}
}

func TestMinimalFlow_Generate_IPv6(t *testing.T) {
	t.Parallel()

	session := NewSession()
	srcIP := net.ParseIP("2001:db8::1")
	dstIP := net.ParseIP("2001:db8::2")

	mf := new(MinimalFlow).Generate(srcIP, dstIP, utils.HTTPSPort, session)

	// IPv4 fields should be zeroed for IPv6
	if mf.SrcAddr != 0 {
		t.Errorf("expected zeroed IPv4 src, got %d", mf.SrcAddr)
	}
	if mf.DstAddr != 0 {
		t.Errorf("expected zeroed IPv4 dst, got %d", mf.DstAddr)
	}
}

func TestExtendedFlow_Generate_IPv6(t *testing.T) {
	t.Parallel()

	session := NewSession()
	srcIP := net.ParseIP("2001:db8::1")
	dstIP := net.ParseIP("2001:db8::2")

	ef := new(ExtendedFlow).Generate(srcIP, dstIP, utils.HTTPSPort, session)

	// IPv4 fields should be zeroed for IPv6
	if ef.SrcAddr != 0 {
		t.Errorf("expected zeroed IPv4 src, got %d", ef.SrcAddr)
	}
	if ef.DstAddr != 0 {
		t.Errorf("expected zeroed IPv4 dst, got %d", ef.DstAddr)
	}
}
