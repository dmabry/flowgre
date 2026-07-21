// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"fmt"
	"net"
	"time"

	"github.com/dmabry/flowgre/utils"
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
	POSTIP_DIFF_SERV_CODE_POINT  = 98
	REPLICATION_FACTOR           = 99
	LAYER2_PKT_SECTION_OFFSET    = 102
	LAYER2_PKT_SECTION_SIZE      = 103
	LAYER2_PKT_SECTION_DATA      = 104
)

// GenericFlow is used to create and generate NetFlow v9 flow records.
// Field order must match GetTemplateFields() exactly — binary.Write
// serializes in struct field order, and the template defines the wire format.
type GenericFlow struct {
	InBytes       uint32
	OutBytes      uint32
	InPkts        uint32
	OutPkts       uint32
	Ipv4SrcAddr   uint32
	Ipv4DstAddr   uint32
	Ipv6SrcAddr   [16]byte
	Ipv6DstAddr   [16]byte
	Ipv6SrcMask   uint8
	Ipv6DstMask   uint8
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
// Field order must match GenericFlow struct field order exactly.
func (gf *GenericFlow) GetTemplateFields() []Field {
	fields := make([]Field, 18)
	fields[0] = Field{Type: IN_BYTES, Length: 4}
	fields[1] = Field{Type: OUT_BYTES, Length: 4}
	fields[2] = Field{Type: IN_PKTS, Length: 4}
	fields[3] = Field{Type: OUT_PKTS, Length: 4}
	fields[4] = Field{Type: IPV4_SRC_ADDR, Length: 4}
	fields[5] = Field{Type: IPV4_DST_ADDR, Length: 4}
	fields[6] = Field{Type: IPV6_SRC_ADDR, Length: 16}
	fields[7] = Field{Type: IPV6_DST_ADDR, Length: 16}
	fields[8] = Field{Type: IPV6_SRC_MASK, Length: 1}
	fields[9] = Field{Type: IPV6_DST_MASK, Length: 1}
	fields[10] = Field{Type: L4_SRC_PORT, Length: 2}
	fields[11] = Field{Type: L4_DST_PORT, Length: 2}
	fields[12] = Field{Type: PROTOCOL, Length: 1}
	fields[13] = Field{Type: TCP_FLAGS, Length: 1}
	fields[14] = Field{Type: FIRST_SWITCHED, Length: 4}
	fields[15] = Field{Type: LAST_SWITCHED, Length: 4}
	fields[16] = Field{Type: ENGINE_TYPE, Length: 1}
	fields[17] = Field{Type: ENGINE_ID, Length: 1}
	return fields
}

// Generate returns a NetFlow v9 Flow with randomly generated payload.
// Populates both IPv4 and IPv6 fields based on the input IP version.
func (gf *GenericFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *Session) (GenericFlow, error) {
	now := time.Now().UnixNano()
	startTime := session.StartTime()
	uptime := uint32((now-startTime)/int64(time.Millisecond)) + 1000
	var err error
	gf.InBytes, err = utils.GenerateRand32(10000)
	if err != nil {
		return GenericFlow{}, fmt.Errorf("generate InBytes: %w", err)
	}
	gf.OutBytes, err = utils.GenerateRand32(10000)
	if err != nil {
		return GenericFlow{}, fmt.Errorf("generate OutBytes: %w", err)
	}
	gf.InPkts, err = utils.GenerateRand32(10000)
	if err != nil {
		return GenericFlow{}, fmt.Errorf("generate InPkts: %w", err)
	}
	gf.OutPkts, err = utils.GenerateRand32(10000)
	if err != nil {
		return GenericFlow{}, fmt.Errorf("generate OutPkts: %w", err)
	}

	// Populate IP addresses based on version
	if srcIP.To4() != nil {
		gf.Ipv4SrcAddr = utils.IPToNum(srcIP)
		gf.Ipv4DstAddr = utils.IPToNum(dstIP)
		gf.Ipv6SrcAddr = [16]byte{}
		gf.Ipv6DstAddr = [16]byte{}
		gf.Ipv6SrcMask = 0
		gf.Ipv6DstMask = 0
	} else {
		gf.Ipv4SrcAddr = 0
		gf.Ipv4DstAddr = 0
		copy(gf.Ipv6SrcAddr[:], srcIP.To16())
		copy(gf.Ipv6DstAddr[:], dstIP.To16())
		gf.Ipv6SrcMask = 64 // Default /64 mask
		gf.Ipv6DstMask = 64 // Default /64 mask
	}

	gf.L4SrcPort, err = utils.GenerateRand16(10000)
	if err != nil {
		return GenericFlow{}, fmt.Errorf("generate L4SrcPort: %w", err)
	}
	tcpFlags, err := utils.RandomNum(0, 32)
	if err != nil {
		return GenericFlow{}, fmt.Errorf("generate TcpFlags: %w", err)
	}
	gf.TcpFlags = uint8(tcpFlags)
	gf.FirstSwitched = uptime - 100
	gf.LastSwitched = uptime - 10
	gf.EngineType = 0
	gf.EngineID = 0

	gf.L4DstPort, gf.Protocol = utils.ResolvePortProtocol(flowSrcPort)

	return *gf, nil
}
