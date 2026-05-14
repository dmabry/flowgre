// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"net"
	"time"

	"github.com/dmabry/flowgre/utils"
)

// ExtendedProfile generates a flow with MAC addresses, VLANs, TTL, and interface info.
type ExtendedProfile struct{}

// Name returns the profile name.
func (p *ExtendedProfile) Name() string { return "extended" }

// TemplateFields returns the 14-field extended template.
func (p *ExtendedProfile) TemplateFields() []Field {
	return []Field{
		{Type: IN_BYTES, Length: 4},
		{Type: IN_PKTS, Length: 4},
		{Type: IPV4_SRC_ADDR, Length: 4},
		{Type: IPV4_DST_ADDR, Length: 4},
		{Type: L4_SRC_PORT, Length: 2},
		{Type: L4_DST_PORT, Length: 2},
		{Type: PROTOCOL, Length: 1},
		{Type: IN_SRC_MAC, Length: 6},
		{Type: OUT_DST_MAC, Length: 6},
		{Type: SRC_VLAN, Length: 2},
		{Type: DST_VLAN, Length: 2},
		{Type: MIN_TTL, Length: 1},
		{Type: MAX_TTL, Length: 1},
		{Type: FIRST_SWITCHED, Length: 4},
		{Type: LAST_SWITCHED, Length: 4},
	}
}

// ExtendedFlow is an extended NetFlow v9 flow record with 15 fields.
// Field order must match ExtendedProfile.TemplateFields() exactly.
type ExtendedFlow struct {
	InBytes       uint32
	InPkts        uint32
	SrcAddr       uint32
	DstAddr       uint32
	SrcPort       uint16
	DstPort       uint16
	Protocol      uint8
	SrcMac        [6]byte
	DstMac        [6]byte
	SrcVlan       uint16
	DstVlan       uint16
	MinTtl        uint8
	MaxTtl        uint8
	FirstSwitched uint32
	LastSwitched  uint32
}

// Generate creates an ExtendedFlow with randomly generated data.
func (ef *ExtendedFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *Session) ExtendedFlow {
	now := time.Now().UnixNano()
	startTime := session.StartTime()
	uptime := uint32((now-startTime)/int64(time.Millisecond)) + 1000

	ef.InBytes = utils.GenerateRand32(10000)
	ef.InPkts = utils.GenerateRand32(10000)

	if srcIP.To4() != nil {
		ef.SrcAddr = utils.IPToNum(srcIP)
		ef.DstAddr = utils.IPToNum(dstIP)
	} else {
		ef.SrcAddr = 0
		ef.DstAddr = 0
	}

	ef.SrcPort = utils.GenerateRand16(10000)
	ef.SrcMac = [6]byte{
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
	}
	ef.DstMac = [6]byte{
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
		uint8(utils.RandomNum(0, 256)),
	}
	ef.SrcVlan = uint16(utils.RandomNum(1, 4094))
	ef.DstVlan = uint16(utils.RandomNum(1, 4094))
	ef.MinTtl = uint8(utils.RandomNum(1, 128))
	ef.MaxTtl = uint8(utils.RandomNum(1, 128))
	ef.FirstSwitched = uptime - 100
	ef.LastSwitched = uptime - 10
	ef.Protocol = uint8(tcpProto)

	switch flowSrcPort {
	case sshPort:
		ef.DstPort = uint16(sshPort)
		ef.Protocol = uint8(tcpProto)
	case ftpPort:
		ef.DstPort = uint16(ftpPort)
		ef.Protocol = uint8(tcpProto)
	case dnsPort:
		ef.DstPort = uint16(dnsPort)
		ef.Protocol = uint8(udpProto)
	case httpPort:
		ef.DstPort = uint16(httpPort)
		ef.Protocol = uint8(tcpProto)
	case httpsPort:
		ef.DstPort = uint16(httpsPort)
		ef.Protocol = uint8(tcpProto)
	case ntpPort:
		ef.DstPort = uint16(ntpPort)
		ef.Protocol = uint8(udpProto)
	case snmpPort:
		ef.DstPort = uint16(snmpPort)
		ef.Protocol = uint8(udpProto)
	case imapsPort:
		ef.DstPort = uint16(imapsPort)
		ef.Protocol = uint8(tcpProto)
	case mysqlPort:
		ef.DstPort = uint16(mysqlPort)
		ef.Protocol = uint8(tcpProto)
	case httpAltPort:
		ef.DstPort = uint16(httpAltPort)
		ef.Protocol = uint8(tcpProto)
	case httpsAltPort:
		ef.DstPort = uint16(httpsAltPort)
		ef.Protocol = uint8(tcpProto)
	case p2pPort:
		ef.DstPort = uint16(p2pPort)
		ef.Protocol = uint8(tcpProto)
	case btPort:
		ef.DstPort = uint16(btPort)
		ef.Protocol = uint8(tcpProto)
	default:
		ef.DstPort = uint16(httpsPort)
		ef.Protocol = uint8(tcpProto)
	}

	return *ef
}
