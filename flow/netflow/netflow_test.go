// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"bytes"
	"encoding/binary"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestHeader_Generate(t *testing.T) {
	flowCount := 10
	sourceID := 618
	ft := new(FlowTracker).Init()
	header := new(Header).Generate(flowCount, sourceID, &ft)

	if header.Version != 9 {
		t.Errorf("Header returned the wrong version! Got: %d Want: 9", header.Version)
	}
	if header.SysUptime == 0 {
		t.Errorf("Header returned the wrong system uptime! Got: %d Want: value", header.SysUptime)
	}
	if header.FlowCount != uint16(flowCount) {
		t.Errorf("Header returned the wrong flow count! Got: %d Want: %d", header.FlowCount, flowCount)
	}
	if header.UnixSec == 0 {
		t.Errorf("Header returned the wrong unix seconds! Got: %d Want: value", header.UnixSec)
	}
	if header.FlowSequence == 0 {
		t.Errorf("Header returned the wrong flow sequence! Got: %d Want: value", header.FlowSequence)
	}
	if header.SourceID != uint32(sourceID) {
		t.Errorf("Header returned the wrong source id! Got: %d Want: %d", header.SourceID, sourceID)
	}
}

func TestGenerateTemplateNetflow(t *testing.T) {
	sourceID := 618
	ft := new(FlowTracker).Init()
	flow := GenerateTemplateNetflow(sourceID, &ft)
	if len(flow.TemplateFlowSets) < 1 {
		t.Errorf("Returned incorrect number of Template Flows! Got: %d Want: >1", len(flow.TemplateFlowSets))
	} else {
		for _, tFlow := range flow.TemplateFlowSets {
			if tFlow.FlowSetID != 0 {
				t.Errorf("Returned wrong FlowSetID! Got: %d Want: %d", tFlow.FlowSetID, 0)
			}
			if tFlow.Length != 64 {
				t.Errorf("Returned wrong length! Got: %d Want: %d", tFlow.Length, 64)
			}
		}
	}
}

func TestGenerateDataNetflow(t *testing.T) {
	flowcount := 10
	sourceID := 618
	ft := new(FlowTracker).Init()
	flow := GenerateDataNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", &ft)

	if len(flow.DataFlowSets) < 1 {
		t.Errorf("Returned incorrect number of Data Flows! Got: %d Want >: %d", len(flow.DataFlowSets), 1)
	} else {
		for _, dFlow := range flow.DataFlowSets {
			if dFlow.FlowSetID < 256 {
				t.Errorf("Returned invalid FlowSetID! Got: %d Want >: %d", dFlow.FlowSetID, 256)
			}
			if dFlow.Length < 64 {
				t.Errorf("Returned invalid length for given parameters! Got: %d Want: %d", dFlow.Length, 64)
			}
			if len(dFlow.Items) < flowcount {
				t.Errorf("Returned invalid number of flows! Got: %d Want: %d", len(dFlow.Items), flowcount)
			}
		}
	}
}

func TestToBytes(t *testing.T) {
	// Generate Netflow Data
	sourceID := 618
	flowcount := 10
	ft := new(FlowTracker).Init()
	tflow := GenerateTemplateNetflow(sourceID, &ft)
	dflow := GenerateDataNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", &ft)
	// Convert to Bytes
	tbuf := tflow.ToBytes()
	dbuf := dflow.ToBytes()
	// Verify Bytes match generated Netflow Data
	tLength := tbuf.Len()
	dLength := dbuf.Len()
	tread := make([]byte, tLength)
	tn, err := tbuf.Read(tread)
	if err != nil {
		t.Errorf("Error during NetFlow Template Read! Got: %v", err)
	}
	if tn != tLength {
		t.Errorf("Returned invalid NetFlow Template buffer length! Got: %d Want: %d", tn, tbuf.Len())
	}
	dread := make([]byte, dLength)
	dn, err := dbuf.Read(dread)
	if err != nil {
		t.Errorf("Error during NetFlow Data Read! Got: %v", err)
	}
	if dn != dLength {
		t.Errorf("Returned invalid Netflow Data buffer length! Got: %d Want: %d", dn, dbuf.Len())
	}
	// Create readers and parse into new netflow structs
	tparsed := Netflow{}
	dparsed := Netflow{}
	treader := bytes.NewReader(tread)
	dreader := bytes.NewReader(dread)
	// Parse Template Header
	err = binary.Read(treader, binary.BigEndian, &tparsed.Header)
	if err != nil {
		t.Errorf("Failed to parse Netflow Header! Got: %v", err)
	}
	if int(tparsed.Header.SourceID) != sourceID {
		t.Errorf("Failed to parse Netflow Header Source ID! Got: %d Want: %d",
			int(tparsed.Header.SourceID), sourceID)
	}
	// Parse TemplateFlow
	tFlowCount := int(tparsed.Header.FlowCount)
	for i := 0; i < tFlowCount; i++ {
		tFlowSet := new(TemplateFlowSet)
		template := new(Template)
		err := binary.Read(treader, binary.BigEndian, &tFlowSet.FlowSetID)
		if err != nil {
			t.Errorf("Failed to parse Netflow FlowSetID! Got: %v", err)
		}
		err = binary.Read(treader, binary.BigEndian, &tFlowSet.Length)
		if err != nil {
			t.Errorf("Failed to parse Netflow FlowSet Length! Got: %v", err)
		}
		err = binary.Read(treader, binary.BigEndian, &template.TemplateID)
		if err != nil {
			t.Errorf("Failed to parse Netflow FlowSet TemplateID! Got: %v", err)
		}
		err = binary.Read(treader, binary.BigEndian, &template.FieldCount)
		if err != nil {
			t.Errorf("Failed to parse Netflow FlowSet FieldCount! Got: %v", err)
		}
		fc := int(template.FieldCount)
		for f := 0; f < fc; f++ {
			tField := new(Field)
			err := binary.Read(treader, binary.BigEndian, &tField.Type)
			if err != nil {
				t.Errorf("Failed to parse Netflow FlowSet Field Type! Got: %v", err)
			}
			err = binary.Read(treader, binary.BigEndian, &tField.Length)
			if err != nil {
				t.Errorf("Failed to parse Netflow FlowSet Field Length! Got: %v", err)
			}
			template.Fields = append(template.Fields, *tField)
		}
		tFlowSet.Templates = append(tFlowSet.Templates, *template)
		tparsed.TemplateFlowSets = append(tparsed.TemplateFlowSets, *tFlowSet)
		t.Log("Completed Template Flow parsing successfully")
	}
	// Parse Data Header
	err = binary.Read(dreader, binary.BigEndian, &dparsed.Header)
	if err != nil {
		t.Errorf("Failed to parse Netflow Header! Got: %v", err)
	}
	if int(dparsed.Header.SourceID) != sourceID {
		t.Errorf("Failed to parse Netflow Header Source ID! Got: %d Want: %d",
			int(dparsed.Header.SourceID), sourceID)
	}
	// Parse DataFlows
	dFlowSet := new(DataFlowSet)
	err = binary.Read(dreader, binary.BigEndian, &dFlowSet.FlowSetID)
	if err != nil {
		t.Errorf("Failed to parse Netflow Data FlowSetID! Got: %v", err)
	}
	err = binary.Read(dreader, binary.BigEndian, &dFlowSet.Length)
	if err != nil {
		t.Errorf("Failed to parse Netflow Data Length! Got: %v", err)
	}
	// I know the field count from the template generated above.  Going to use that
	dataItems := make([]DataAny, flowcount)
	for i := 0; i < flowcount; i++ {
		dataItem := GenericFlow{}
		err := binary.Read(dreader, binary.BigEndian, &dataItem)
		if err != nil {
			t.Errorf("Issue reading in GenericFlow")
		}
		dataItems[i] = dataItem
	}
	dFlowSet.Items = dataItems
	if dreader.Len() > 0 {
		// read the rest as padding
		padLength := dreader.Len()
		padding := make([]byte, padLength)
		err := binary.Read(dreader, binary.BigEndian, padding)
		if err != nil {
			t.Errorf("Failed to parse Netflow Data Padding! Got: %v", err)
		}
		dFlowSet.Padding = padLength
	}
	dparsed.DataFlowSets = append(dparsed.DataFlowSets, *dFlowSet)
	t.Log("Completed Data Flow parsing successfully")
	// make sure they are equal
	if !cmp.Equal(tflow, tparsed) {
		t.Error("Failed Generated Netflow Template Flow and Parsed is different!")
	} else {
		t.Log("Generated Netflow Template Flow and Parsed Match!")
	}
	if !cmp.Equal(dflow, dparsed) {
		t.Error("Failed Generated Netflow Data Flow and Parsed is different!")
	} else {
		t.Log("Generated Netflow Data Flow and Parsed Match!")
	}
}
