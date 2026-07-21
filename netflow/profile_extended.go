// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"fmt"
	"net"
	"time"

	"github.com/dmabry/flowgre/utils"
)

// ExtendedProfile generates a flow with MAC addresses, VLANs, TTL, and interface info.
type ExtendedProfile struct{}

// Name returns the profile name.
func (p *ExtendedProfile) Name() string { return "extended" }

// TemplateFields returns the 15-field extended template.
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
func (ef *ExtendedFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *Session) (ExtendedFlow, error) {
	now := time.Now().UnixNano()
	startTime := session.StartTime()
	uptime := uint32((now-startTime)/int64(time.Millisecond)) + 1000

	var err error
	ef.InBytes, err = utils.GenerateRand32(10000)
	if err != nil {
		return ExtendedFlow{}, fmt.Errorf("generate InBytes: %w", err)
	}
	ef.InPkts, err = utils.GenerateRand32(10000)
	if err != nil {
		return ExtendedFlow{}, fmt.Errorf("generate InPkts: %w", err)
	}

	if srcIP.To4() != nil {
		ef.SrcAddr = utils.IPToNum(srcIP)
		ef.DstAddr = utils.IPToNum(dstIP)
	} else {
		ef.SrcAddr = 0
		ef.DstAddr = 0
	}

	ef.SrcPort, err = utils.GenerateRand16(10000)
	if err != nil {
		return ExtendedFlow{}, fmt.Errorf("generate SrcPort: %w", err)
	}
	ef.SrcMac = [6]byte{}
	for i := range ef.SrcMac {
		val, err := utils.RandomNum(0, 256)
		if err != nil {
			return ExtendedFlow{}, fmt.Errorf("generate SrcMac byte %d: %w", i, err)
		}
		ef.SrcMac[i] = uint8(val)
	}
	ef.DstMac = [6]byte{}
	for i := range ef.DstMac {
		val, err := utils.RandomNum(0, 256)
		if err != nil {
			return ExtendedFlow{}, fmt.Errorf("generate DstMac byte %d: %w", i, err)
		}
		ef.DstMac[i] = uint8(val)
	}
	srcVlan, err := utils.RandomNum(1, 4094)
	if err != nil {
		return ExtendedFlow{}, fmt.Errorf("generate SrcVlan: %w", err)
	}
	ef.SrcVlan = uint16(srcVlan)
	dstVlan, err := utils.RandomNum(1, 4094)
	if err != nil {
		return ExtendedFlow{}, fmt.Errorf("generate DstVlan: %w", err)
	}
	ef.DstVlan = uint16(dstVlan)
	minTtl, err := utils.RandomNum(1, 128)
	if err != nil {
		return ExtendedFlow{}, fmt.Errorf("generate MinTtl: %w", err)
	}
	ef.MinTtl = uint8(minTtl)
	maxTtl, err := utils.RandomNum(1, 128)
	if err != nil {
		return ExtendedFlow{}, fmt.Errorf("generate MaxTtl: %w", err)
	}
	ef.MaxTtl = uint8(maxTtl)
	ef.FirstSwitched = uptime - 100
	ef.LastSwitched = uptime - 10

	ef.DstPort, ef.Protocol = utils.ResolvePortProtocol(flowSrcPort)

	return *ef, nil
}
