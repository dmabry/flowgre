// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"net"
	"time"

	"github.com/dmabry/flowgre/utils"
)

// Port constants
const (
	ftpPort      = 21
	sshPort      = 22
	dnsPort      = 53
	httpPort     = 80
	httpsPort    = 443
	ntpPort      = 123
	snmpPort     = 161
	imapsPort    = 993
	mysqlPort    = 3306
	httpAltPort  = 8080
	httpsAltPort = 8443
	p2pPort      = 6681
	btPort       = 6682
)

// Protocol constants
const (
	tcpProto   = 6
	udpProto   = 17
	icmpProto  = 1
	sctpProto  = 132
	igmpProto  = 2
	egpProto   = 8
	igpProto   = 9
	greProto   = 47
	espProto   = 50
	eigrpProto = 88
)

// Constants for Field Types
const (
	IN_BYTES                     = 1
	IN_PKTS                      = 2
	FLOWS                        = 3
	PROTOCOL                     = 4
	SRC_TOS                      = 5
	TCP_FLAGS                    = 6
	L4_SRC_PORT                  = 7
	IPV4_SRC_ADDR                = 8
	SRC_MASK                     = 9
	INPUT_SNMP                   = 10
	L4_DST_PORT                  = 11
	IPV4_DST_ADDR                = 12
	DST_MASK                     = 13
	OUTPUT_SNMP                  = 14
	IPV4_NEXT_HOP                = 15
	SRC_AS                       = 16
	DST_AS                       = 17
	BGP_IPV4_NEXT_HOP            = 18
	MUL_DST_PKTS                 = 19
	MUL_DST_BYTES                = 20
	LAST_SWITCHED                = 21
	FIRST_SWITCHED               = 22
	OUT_BYTES                    = 23
	OUT_PKTS                     = 24
	MIN_PKT_LNGTH                = 25
	MAX_PKT_LNGTH                = 26
	IPV6_SRC_ADDR                = 27
	IPV6_DST_ADDR                = 28
	IPV6_SRC_MASK                = 29
	IPV6_DST_MASK                = 30
	IPV6_FLOW_LABEL              = 31
	ICMP_TYPE                    = 32
	MUL_IGMP_TYPE                = 33
	SAMPLING_INTERVAL            = 34
	SAMPLING_ALGORITHM           = 35
	FLOW_ACTIVE_TIMEOUT          = 36
	FLOW_INACTIVE_TIMEOUT        = 37
	ENGINE_TYPE                  = 38
	ENGINE_ID                    = 39
	TOTAL_BYTES_EXP              = 40
	TOTAL_PKTS_EXP               = 41
	TOTAL_FLOWS_EXP              = 42
	IPV4_SRC_PREFIX              = 44
	IPV4_DST_PREFIX              = 45
	MPLS_TOP_LABEL_TYPE          = 46
	MPLS_TOP_LABEL_IP_ADDR       = 47
	FLOW_SAMPLER_ID              = 48
	FLOW_SAMPLER_MODE            = 49
	FLOW_SAMPLER_RANDOM_INTERVAL = 50
	MIN_TTL                      = 52
	MAX_TTL                      = 53
	IPV4_IDENT                   = 54
	DST_TOS                      = 55
	IN_SRC_MAC                   = 56
	OUT_DST_MAC                  = 57
	SRC_VLAN                     = 58
	DST_VLAN                     = 59
	IP_PROTOCOL_VERSION          = 60
	DIRECTION                    = 61
	IPV6_NEXT_HOP                = 62
	BGP_IPV6_NEXT_HOP            = 63
	IPV6_OPTION_HEADERS          = 64
	MPLS_LABEL_1                 = 70
	MPLS_LABEL_2                 = 71
	MPLS_LABEL_3                 = 72
	MPLS_LABEL_4                 = 73
	MPLS_LABEL_5                 = 74
	MPLS_LABEL_6                 = 75
	MPLS_LABEL_7                 = 76
	MPLS_LABEL_8                 = 77
	MPLS_LABEL_9                 = 78
	MPLS_LABEL_10                = 79
	IN_DST_MAC                   = 80
	OUT_SRC_MAC                  = 81
	IF_NAME                      = 82
	IF_DESC                      = 83
	SAMPLER_NAME                 = 84
	IN_PERMANENT_BYTES           = 85
	IN_PERMANENT_PKTS            = 86
	FRAGMENT_OFFSET              = 88
	FORWARDING_STATUS            = 89
	MPLS_PAL_RD                  = 90
	MPLS_PREFIX_LEN              = 91
	SRC_TRAFFIC_INDEX            = 92
	DST_TRAFFIC_INDEX            = 93
	APPLICATION_DESCRIPTION      = 94
	APPLICATION_TAG              = 95
	APPLICATION_NAME             = 96
	postipDiffServCodePoint      = 98
	replication_factor           = 99
	layer2packetSectionOffset    = 102
	layer2packetSectionSize      = 103
	layer2packetSectionData      = 104
)

// GenericFlow is used to create and generate HTTPS Flows
type GenericFlow struct {
	InBytes       uint32
	OutBytes      uint32
	InPkts        uint32
	OutPkts       uint32
	Ipv4SrcAddr   uint32
	Ipv4DstAddr   uint32
	L4SrcPort     uint16
	L4DstPort     uint16
	Protocol      uint8
	TcpFlags      uint8
	FirstSwitched uint32
	LastSwitched  uint32
	EngineType    uint8
	EngineID      uint8
}

// GetTemplateFields returns the Fields for the Template to be used.
func (gf *GenericFlow) GetTemplateFields() []Field {
	fields := make([]Field, 14)
	fields[0] = Field{Type: IN_BYTES, Length: 4}
	fields[1] = Field{Type: OUT_BYTES, Length: 4}
	fields[2] = Field{Type: IN_PKTS, Length: 4}
	fields[3] = Field{Type: OUT_PKTS, Length: 4}
	fields[4] = Field{Type: IPV4_SRC_ADDR, Length: 4}
	fields[5] = Field{Type: IPV4_DST_ADDR, Length: 4}
	fields[6] = Field{Type: L4_SRC_PORT, Length: 2}
	fields[7] = Field{Type: L4_DST_PORT, Length: 2}
	fields[8] = Field{Type: PROTOCOL, Length: 1}
	fields[9] = Field{Type: TCP_FLAGS, Length: 1}
	fields[10] = Field{Type: FIRST_SWITCHED, Length: 4}
	fields[11] = Field{Type: LAST_SWITCHED, Length: 4}
	fields[12] = Field{Type: ENGINE_TYPE, Length: 1}
	fields[13] = Field{Type: ENGINE_ID, Length: 1}
	return fields
}

// Generate returns HTTPS Flow with randomly generated payload
func (gf *GenericFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *Session) GenericFlow {
	now := time.Now().UnixNano()
	startTime := session.StartTime()
	uptime := uint32((now-startTime)/int64(time.Millisecond)) + 1000
	gf.InBytes = utils.GenerateRand32(10000)
	gf.OutBytes = utils.GenerateRand32(10000)
	gf.InPkts = utils.GenerateRand32(10000)
	gf.OutPkts = utils.GenerateRand32(10000)
	gf.Ipv4SrcAddr = utils.IPToNum(srcIP)
	gf.Ipv4DstAddr = utils.IPToNum(dstIP)
	gf.L4SrcPort = utils.GenerateRand16(10000)
	gf.TcpFlags = uint8(utils.RandomNum(0, 32))
	gf.FirstSwitched = uptime - 100
	gf.LastSwitched = uptime - 10
	gf.EngineType = 0
	gf.EngineID = 0

	switch flowSrcPort {
	case sshPort:
		gf.L4DstPort = uint16(sshPort)
		gf.Protocol = uint8(tcpProto)
	case ftpPort:
		gf.L4DstPort = uint16(ftpPort)
		gf.Protocol = uint8(tcpProto)
	case dnsPort:
		gf.L4DstPort = uint16(dnsPort)
		gf.Protocol = uint8(udpProto)
	case httpPort:
		gf.L4DstPort = uint16(httpPort)
		gf.Protocol = uint8(tcpProto)
	case httpsPort:
		gf.L4DstPort = uint16(httpsPort)
		gf.Protocol = uint8(tcpProto)
	case ntpPort:
		gf.L4DstPort = uint16(ntpPort)
		gf.Protocol = uint8(udpProto)
	case snmpPort:
		gf.L4DstPort = uint16(snmpPort)
		gf.Protocol = uint8(udpProto)
	case imapsPort:
		gf.L4DstPort = uint16(imapsPort)
		gf.Protocol = uint8(tcpProto)
	case mysqlPort:
		gf.L4DstPort = uint16(mysqlPort)
		gf.Protocol = uint8(tcpProto)
	case httpAltPort:
		gf.L4DstPort = uint16(httpAltPort)
		gf.Protocol = uint8(tcpProto)
	case httpsAltPort:
		gf.L4DstPort = uint16(httpsAltPort)
		gf.Protocol = uint8(tcpProto)
	case p2pPort:
		gf.L4DstPort = uint16(p2pPort)
		gf.Protocol = uint8(tcpProto)
	case btPort:
		gf.L4DstPort = uint16(btPort)
		gf.Protocol = uint8(tcpProto)
	default:
		gf.L4DstPort = uint16(httpsPort)
		gf.Protocol = uint8(tcpProto)
	}

	return *gf
}
