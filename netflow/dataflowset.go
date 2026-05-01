// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"encoding/binary"
	"log"

	"github.com/dmabry/flowgre/utils"
)

type DataItem struct {
	Fields []uint32
}

type DataAny interface{}

// DataFlowSet for Netflow
type DataFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Items     []DataAny
	Padding   int
}

// Generate a DataFlowSet.
// Per Netflow v9 spec, FlowSetID is *always* set to the TemplateID from a given TemplateFlowSet.
// Hardcoded TemplateID to 256, but could be variable as long as it is greater than 255
// Currently hardcoded to generate random src/dst IPs from 10.0.0.0/8.
func (d *DataFlowSet) Generate(flowCount int, srcRange string, dstRange string, flowSrcPort int, session *Session) DataFlowSet {
	dataFlowSet := new(DataFlowSet)
	dataFlowSet.FlowSetID = 256
	protoPorts := [13]int{21, 22, 53, 80, 443, 123, 161, 993, 3306, 8080, 8443, 6681, 6682}
	items := make([]DataAny, flowCount)
	for i := 0; i < flowCount; i++ {
		srcIP, err := utils.RandomIP(srcRange)
		if err != nil {
			log.Printf("Issue generating IP... proceeding anyway: %v", err)
		}
		dstIP, err := utils.RandomIP(dstRange)
		if err != nil {
			log.Printf("Issue generating IP... proceeding anyway: %v", err)
		}
		hf := new(GenericFlow)
		var flowPort int
		if flowSrcPort == 0 {
			flowPort = protoPorts[utils.RandomNum(0, 12)]
		} else {
			flowPort = flowSrcPort
		}
		items[i] = hf.Generate(srcIP, dstIP, flowPort, session)
	}
	dataFlowSet.Items = items
	dataFlowSet.Length = uint16(dataFlowSet.size())
	return *dataFlowSet
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
	size += padding      // number of uint8 to pad in order to reach 32 bit boundary
	d.Padding = padding // save the padding as an int for later.
	return size
}
