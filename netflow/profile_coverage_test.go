// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"net"
	"testing"
)

func TestMinimalFlow_Generate_AllProtocols(t *testing.T) {
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

	cases := []struct {
		port         int
		wantPort     uint16
		wantProtocol uint8
	}{
		{sshPort, uint16(sshPort), tcpProto},
		{ftpPort, uint16(ftpPort), tcpProto},
		{dnsPort, uint16(dnsPort), udpProto},
		{httpPort, uint16(httpPort), tcpProto},
		{httpsPort, uint16(httpsPort), tcpProto},
		{ntpPort, uint16(ntpPort), udpProto},
		{snmpPort, uint16(snmpPort), udpProto},
		{imapsPort, uint16(imapsPort), tcpProto},
		{mysqlPort, uint16(mysqlPort), tcpProto},
		{httpAltPort, uint16(httpAltPort), tcpProto},
		{httpsAltPort, uint16(httpsAltPort), tcpProto},
		{p2pPort, uint16(p2pPort), tcpProto},
		{btPort, uint16(btPort), tcpProto},
		{99999, uint16(httpsPort), tcpProto}, // default
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
		{sshPort, uint16(sshPort), tcpProto},
		{ftpPort, uint16(ftpPort), tcpProto},
		{dnsPort, uint16(dnsPort), udpProto},
		{httpPort, uint16(httpPort), tcpProto},
		{httpsPort, uint16(httpsPort), tcpProto},
		{ntpPort, uint16(ntpPort), udpProto},
		{snmpPort, uint16(snmpPort), udpProto},
		{imapsPort, uint16(imapsPort), tcpProto},
		{mysqlPort, uint16(mysqlPort), tcpProto},
		{httpAltPort, uint16(httpAltPort), tcpProto},
		{httpsAltPort, uint16(httpsAltPort), tcpProto},
		{p2pPort, uint16(p2pPort), tcpProto},
		{btPort, uint16(btPort), tcpProto},
		{99999, uint16(httpsPort), tcpProto}, // default
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

	mf := new(MinimalFlow).Generate(srcIP, dstIP, httpsPort, session)

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

	ef := new(ExtendedFlow).Generate(srcIP, dstIP, httpsPort, session)

	// IPv4 fields should be zeroed for IPv6
	if ef.SrcAddr != 0 {
		t.Errorf("expected zeroed IPv4 src, got %d", ef.SrcAddr)
	}
	if ef.DstAddr != 0 {
		t.Errorf("expected zeroed IPv4 dst, got %d", ef.DstAddr)
	}
}
