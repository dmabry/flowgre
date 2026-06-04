// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Netflow v9 funcs and structs used for generating netflow packet to be put on the wire

package netflow

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/dmabry/flowgre/utils"
)

func GenerateNetflow(flowCount int, sourceID int, srcRange string, dstRange string, session *Session, profile ...FlowProfile) (Netflow, error) {
	netflow := new(Netflow)
	templateFlow := new(TemplateFlowSet).Generate(session, profile...)
	dataFlow, err := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, utils.HTTPSPort, session, profile...)
	if err != nil {
		return Netflow{}, fmt.Errorf("generate data flow set: %w", err)
	}
	header := new(Header).Generate(flowCount+1, sourceID, session) // always +1 of dataflow count, because we are counting the template
	netflow.Header = header
	netflow.TemplateFlowSets = append(netflow.TemplateFlowSets, templateFlow)
	netflow.DataFlowSets = append(netflow.DataFlowSets, dataFlow)
	return *netflow, nil
}

// GenerateDataNetflow Generates a Netflow containing Data flows
func GenerateDataNetflow(flowCount int, sourceID int, srcRange string, dstRange string, flowSrcPort int, session *Session, profile ...FlowProfile) (Netflow, error) {
	netflow := new(Netflow)
	dataFlow, err := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, flowSrcPort, session, profile...)
	if err != nil {
		return Netflow{}, fmt.Errorf("generate data flow set: %w", err)
	}
	header := new(Header).Generate(1, sourceID, session) // always 1 for but could be more in future
	netflow.Header = header
	netflow.DataFlowSets = append(netflow.DataFlowSets, dataFlow)
	return *netflow, nil
}

// GenerateTemplateNetflow Generates a Netflow containing Template flow
func GenerateTemplateNetflow(sourceID int, session *Session, profile ...FlowProfile) Netflow {
	netflow := new(Netflow)
	templateFlow := new(TemplateFlowSet).Generate(session, profile...)
	header := new(Header).Generate(1, sourceID, session) // always 1 counting the template only
	netflow.Header = header
	netflow.TemplateFlowSets = append(netflow.TemplateFlowSets, templateFlow)
	return *netflow
}

// IsValidNetFlow validates that the given payload has a netflow v9 header
func IsValidNetFlow(payload []byte, nfVersion int) (bool, error) {
	// yes = true, no = false
	header := Header{}
	reader := bytes.NewReader(payload)
	// Parse Netflow Header
	err := binary.Read(reader, binary.BigEndian, &header)
	if err != nil {
		return false, err
	}
	if header.Version != uint16(nfVersion) {
		return false, fmt.Errorf("Header version doesn't match!  Got %d and expected %d", header.Version, nfVersion)
	}
	return true, nil
}

// UpdateTimeStamp will change the time to current timestamp
func UpdateTimeStamp(payload []byte) ([]byte, error) {
	header := Header{}
	reader := bytes.NewReader(payload)
	err := binary.Read(reader, binary.BigEndian, &header)
	if err != nil {
		return nil, err
	}
	remainder := make([]byte, len(payload)-20) // header is always 20 bytes long
	err = binary.Read(reader, binary.BigEndian, &remainder)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixNano()
	secs := now / int64(time.Second)
	header.UnixSec = uint32(secs)
	var buf bytes.Buffer
	err = binary.Write(&buf, binary.BigEndian, header)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&buf, binary.BigEndian, remainder)
	if err != nil {
		return nil, err
	}
	// Success!  Return the new []byte payload
	return buf.Bytes(), nil
}
