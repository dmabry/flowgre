// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"net"

	"github.com/dmabry/flowgre/utils"
)

// MinimalProfile generates a minimal flow with only essential fields:
// src IP, dst IP, src port, dst port, protocol, bytes, packets.
type MinimalProfile struct{}

// Name returns the profile name.
func (p *MinimalProfile) Name() string { return "minimal" }

// TemplateFields returns the 7-field minimal template.
func (p *MinimalProfile) TemplateFields() []Field {
	return []Field{
		{Type: IN_BYTES, Length: 4},
		{Type: IN_PKTS, Length: 4},
		{Type: IPV4_SRC_ADDR, Length: 4},
		{Type: IPV4_DST_ADDR, Length: 4},
		{Type: L4_SRC_PORT, Length: 2},
		{Type: L4_DST_PORT, Length: 2},
		{Type: PROTOCOL, Length: 1},
	}
}

// MinimalFlow is a minimal NetFlow v9 flow record with 7 essential fields.
// Field order must match MinimalProfile.TemplateFields() exactly.
type MinimalFlow struct {
	InBytes  uint32
	InPkts   uint32
	SrcAddr  uint32
	DstAddr  uint32
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8
}

// Generate creates a MinimalFlow with randomly generated data.
func (mf *MinimalFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *Session) MinimalFlow {
	mf.InBytes = utils.GenerateRand32(10000)
	mf.InPkts = utils.GenerateRand32(10000)

	if srcIP.To4() != nil {
		mf.SrcAddr = utils.IPToNum(srcIP)
		mf.DstAddr = utils.IPToNum(dstIP)
	} else {
		mf.SrcAddr = 0
		mf.DstAddr = 0
	}

	mf.SrcPort = utils.GenerateRand16(10000)

	mf.DstPort, mf.Protocol = utils.ResolvePortProtocol(flowSrcPort)

	return *mf
}
