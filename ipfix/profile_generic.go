// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package ipfix

import (
	"net"

	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

// IPFIXFlowProfile defines an IPFIX flow type with its template fields.
type IPFIXFlowProfile interface {
	// TemplateFields returns the field definitions for the IPFIX template.
	TemplateFields() []Field

	// Name returns a human-readable name for logging.
	Name() string
}

// GenericIPFIXProfile implements IPFIXFlowProfile for the default 19-field IPFIX flow.
type GenericIPFIXProfile struct{}

// Name returns the profile name.
func (p *GenericIPFIXProfile) Name() string {
	return "generic"
}

// TemplateFields returns the 19-field IPFIX template that matches GenericFlow struct layout.
func (p *GenericIPFIXProfile) TemplateFields() []Field {
	return []Field{
		{Type: InOctets, Length: 4},
		{Type: OutOctets, Length: 4},
		{Type: InPackets, Length: 4},
		{Type: OutPackets, Length: 4},
		{Type: SourceIPv4Address, Length: 4},
		{Type: DestinationIPv4Address, Length: 4},
		{Type: SourceIPv6Address, Length: 16},
		{Type: DestinationIPv6Address, Length: 16},
		{Type: SourceIPv6PrefixLength, Length: 1},
		{Type: DestinationIPv6PrefixLength, Length: 1},
		{Type: SourceTransportPort, Length: 2},
		{Type: DestinationTransportPort, Length: 2},
		{Type: ProtocolIdentifier, Length: 1},
		{Type: TCPFlags, Length: 1},
		{Type: FlowStartMilliseconds, Length: 4},
		{Type: FlowEndMilliseconds, Length: 4},
		{Type: FlowDirection, Length: 1},
		{Type: IPClassOfService, Length: 1},
		{Type: FlowEndReason, Length: 1},
	}
}

// MinimalIPFIXProfile generates a minimal IPFIX flow with essential fields.
type MinimalIPFIXProfile struct{}

// Name returns the profile name.
func (p *MinimalIPFIXProfile) Name() string { return "minimal" }

// TemplateFields returns the 7-field minimal IPFIX template.
func (p *MinimalIPFIXProfile) TemplateFields() []Field {
	return []Field{
		{Type: InOctets, Length: 4},
		{Type: InPackets, Length: 4},
		{Type: SourceIPv4Address, Length: 4},
		{Type: DestinationIPv4Address, Length: 4},
		{Type: SourceTransportPort, Length: 2},
		{Type: DestinationTransportPort, Length: 2},
		{Type: ProtocolIdentifier, Length: 1},
	}
}

// MinimalIPFIXFlow is a minimal IPFIX flow record with 7 essential fields.
type MinimalIPFIXFlow struct {
	InOctets           uint32
	InPackets          uint32
	SourceIPv4Addr     uint32
	DestIPv4Addr       uint32
	SourcePort         uint16
	DestPort           uint16
	ProtocolIdentifier uint8
}

// Generate creates a MinimalIPFIXFlow with randomly generated data.
func (mf *MinimalIPFIXFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *netflow.Session) MinimalIPFIXFlow {
	mf.InOctets = utils.GenerateRand32(10000)
	mf.InPackets = utils.GenerateRand32(10000)

	if srcIP.To4() != nil {
		mf.SourceIPv4Addr = utils.IPToNum(srcIP)
		mf.DestIPv4Addr = utils.IPToNum(dstIP)
	} else {
		mf.SourceIPv4Addr = 0
		mf.DestIPv4Addr = 0
	}

	mf.SourcePort = utils.GenerateRand16(10000)
	mf.ProtocolIdentifier = utils.TCPProto

	switch flowSrcPort {
	case utils.SSHPort:
		mf.DestPort = uint16(utils.SSHPort)
		mf.ProtocolIdentifier = utils.TCPProto
	case utils.FTPPort:
		mf.DestPort = uint16(utils.FTPPort)
		mf.ProtocolIdentifier = utils.TCPProto
	case utils.DNSPort:
		mf.DestPort = uint16(utils.DNSPort)
		mf.ProtocolIdentifier = utils.UDPProto
	case utils.HTTPPort:
		mf.DestPort = uint16(utils.HTTPPort)
		mf.ProtocolIdentifier = utils.TCPProto
	case utils.HTTPSPort:
		mf.DestPort = uint16(utils.HTTPSPort)
		mf.ProtocolIdentifier = utils.TCPProto
	default:
		mf.DestPort = uint16(utils.HTTPSPort)
		mf.ProtocolIdentifier = utils.TCPProto
	}

	return *mf
}

// ExtendedIPFIXProfile generates an extended IPFIX flow with additional fields.
type ExtendedIPFIXProfile struct{}

// Name returns the profile name.
func (p *ExtendedIPFIXProfile) Name() string { return "extended" }

// TemplateFields returns the extended IPFIX template with additional fields.
func (p *ExtendedIPFIXProfile) TemplateFields() []Field {
	return []Field{
		{Type: InOctets, Length: 4},
		{Type: OutOctets, Length: 4},
		{Type: InPackets, Length: 4},
		{Type: OutPackets, Length: 4},
		{Type: SourceIPv4Address, Length: 4},
		{Type: DestinationIPv4Address, Length: 4},
		{Type: SourceTransportPort, Length: 2},
		{Type: DestinationTransportPort, Length: 2},
		{Type: ProtocolIdentifier, Length: 1},
		{Type: TCPFlags, Length: 1},
		{Type: FlowStartMilliseconds, Length: 4},
		{Type: FlowEndMilliseconds, Length: 4},
		{Type: FlowDirection, Length: 1},
		{Type: IPClassOfService, Length: 1},
		{Type: FlowEndReason, Length: 1},
		{Type: SourceIPv6Address, Length: 16},
		{Type: DestinationIPv6Address, Length: 16},
	}
}
