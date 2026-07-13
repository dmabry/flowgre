// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/dmabry/flowgre/utils"
)

// DataFlowSet for Netflow
type DataFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Items     []any
	Padding   int
}

// Generate a DataFlowSet.
// Per Netflow v9 spec, FlowSetID is *always* set to the TemplateID from a given TemplateFlowSet.
// Hardcoded TemplateID to 256, but could be variable as long as it is greater than 255.
// Currently hardcoded to generate random src/dst IPs from 10.0.0.0/8.
// If profile is nil, defaults to GenericProfile for backward compatibility.
func (d *DataFlowSet) Generate(flowCount int, srcRange string, dstRange string, flowSrcPort int, session *Session, profile ...FlowProfile) (DataFlowSet, error) {
	p := FlowProfile(&GenericProfile{}) // default
	if len(profile) > 0 && profile[0] != nil {
		p = profile[0]
	}

	dataFlowSet := new(DataFlowSet)
	dataFlowSet.FlowSetID = 256
	protoPorts := utils.ProtoPorts
	items := make([]any, flowCount)
	for i := range flowCount {
		srcIP, err := utils.RandomIPCIDR(srcRange)
		if err != nil {
			return DataFlowSet{}, fmt.Errorf("failed to generate src IP for flow %d: %w", i, err)
		}
		dstIP, err := utils.RandomIPCIDR(dstRange)
		if err != nil {
			return DataFlowSet{}, fmt.Errorf("failed to generate dst IP for flow %d: %w", i, err)
		}
		var flowPort int
		if flowSrcPort == 0 {
			flowPort = protoPorts[utils.RandomNum(0, len(protoPorts))]
		} else {
			flowPort = flowSrcPort
		}
		items[i] = generateFlow(p, srcIP, dstIP, flowPort, session)
	}
	dataFlowSet.Items = items
	size := dataFlowSet.size()
	if size > 0xFFFF {
		return DataFlowSet{}, fmt.Errorf("DataFlowSet size %d exceeds uint16 max (65535)", size)
	}
	dataFlowSet.Length = uint16(size)
	return *dataFlowSet, nil
}

// generateFlow creates a flow record appropriate for the given profile.
func generateFlow(p FlowProfile, srcIP, dstIP net.IP, flowPort int, session *Session) any {
	switch prof := p.(type) {
	case *MinimalProfile:
		_ = prof
		return new(MinimalFlow).Generate(srcIP, dstIP, flowPort, session)
	case *ExtendedProfile:
		_ = prof
		return new(ExtendedFlow).Generate(srcIP, dstIP, flowPort, session)
	default:
		return new(GenericFlow).Generate(srcIP, dstIP, flowPort, session)
	}
}

// Get the size of the DataFlowSet in bytes
func (d *DataFlowSet) size() int {
	padding := 0
	size := binary.Size(d.FlowSetID)
	size += binary.Size(d.Length)
	for _, item := range d.Items {
		size += binary.Size(item)
	}
	remainder := size % 4
	if remainder > 0 {
		padding = 4 - remainder
	}
	size += padding     // number of uint8 to pad in order to reach 32 bit boundary
	d.Padding = padding // save the padding as an int for later.
	return size
}
