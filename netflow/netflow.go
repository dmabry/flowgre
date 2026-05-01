// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Netflow v9 funcs and structs used for generating netflow packet to be put on the wire

package netflow

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// FlowTracker is deprecated — use *Session instead.
// Kept for API compatibility during refactor.
type FlowTracker struct {
	session *Session
}

// Init creates a new FlowTracker wrapping a fresh Session.
func (ft *FlowTracker) Init() FlowTracker {
	return FlowTracker{session: NewSession()}
}

// GetStartTime returns the wrapped session's start time.
func (ft *FlowTracker) GetStartTime() int64 {
	return ft.session.StartTime()
}

// NextSeq delegates to the wrapped session.
func (ft *FlowTracker) NextSeq() uint32 {
	return ft.session.NextSeq()
}

// GenerateNetflow Generates a combined Template and Data flow Netflow struct.  Not required by spec, but can be done.
func GenerateNetflow(flowCount int, sourceID int, srcRange string, dstRange string, flowTracker *FlowTracker) Netflow {
	session := NewSession()
	netflow := new(Netflow)
	templateFlow := new(TemplateFlowSet).Generate()
	dataFlow := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, httpsPort, session)
	header := new(Header).Generate(flowCount+1, sourceID, session) // always +1 of dataflow count, because we are counting the template
	netflow.Header = header
	netflow.TemplateFlowSets = append(netflow.TemplateFlowSets, templateFlow)
	netflow.DataFlowSets = append(netflow.DataFlowSets, dataFlow)
	return *netflow
}

// GenerateDataNetflow Generates a Netflow containing Data flows
func GenerateDataNetflow(flowCount int, sourceID int, srcRange string, dstRange string, flowSrcPort int, flowTracker *FlowTracker) Netflow {
	session := NewSession()
	netflow := new(Netflow)
	dataFlow := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, flowSrcPort, session)
	header := new(Header).Generate(1, sourceID, session) // always 1 for but could be more in future
	netflow.Header = header
	netflow.DataFlowSets = append(netflow.DataFlowSets, dataFlow)
	return *netflow
}

// GenerateTemplateNetflow Generates a Netflow containing Template flow
func GenerateTemplateNetflow(sourceID int, flowTracker *FlowTracker) Netflow {
	session := NewSession()
	netflow := new(Netflow)
	templateFlow := new(TemplateFlowSet).Generate()
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
