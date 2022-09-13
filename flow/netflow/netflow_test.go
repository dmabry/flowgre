// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import "testing"

func TestHeader_Generate(t *testing.T) {
	flowCount := 10
	sourceID := 618
	header := new(Header).Generate(flowCount, sourceID)

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
	flow := GenerateTemplateNetflow(sourceID)
	if len(flow.TemplateFlowSets) < 1 {
		t.Errorf("Returned incorrect number of Template Flows! Got: %d Want: >1", len(flow.TemplateFlowSets))
	} else {
		for _, tFlow := range flow.TemplateFlowSets {
			if tFlow.FlowSetID != 0 {
				t.Errorf("Returned wrong FlowSetID! Got: %d Want: %d", tFlow.FlowSetID, 0)
			}
			if tFlow.Length != 32 {
				t.Errorf("Returned wrong length! Got: %d Want: %d", tFlow.Length, 32)
			}
		}
	}
}

func TestGenerateDataNetflow(t *testing.T) {
	flowcount := 10
	sourceID := 618
	flow := GenerateDataNetflow(flowcount, sourceID)

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
