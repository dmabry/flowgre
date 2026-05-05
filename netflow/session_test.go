package netflow

import (
	"testing"
)

func TestFlowSequenceMonotonicallyIncreases(t *testing.T) {
	t.Parallel()
	session := NewSession()
	f1 := GenerateDataNetflow(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, session)
	f2 := GenerateDataNetflow(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, session)
	f3 := GenerateDataNetflow(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, session)
	if f1.Header.FlowSequence >= f2.Header.FlowSequence {
		t.Errorf("FlowSequence not monotonically increasing: f1=%d >= f2=%d", f1.Header.FlowSequence, f2.Header.FlowSequence)
	}
	if f2.Header.FlowSequence >= f3.Header.FlowSequence {
		t.Errorf("FlowSequence not monotonically increasing: f2=%d >= f3=%d", f2.Header.FlowSequence, f3.Header.FlowSequence)
	}
}

func TestFlowSequenceResetsPerSession(t *testing.T) {
	t.Parallel()
	s1 := NewSession()
	s2 := NewSession()
	f1 := GenerateDataNetflow(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, s1)
	f2 := GenerateDataNetflow(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, s2)
	if f1.Header.FlowSequence != f2.Header.FlowSequence {
		t.Errorf("Different sessions should start from same sequence: f1=%d f2=%d", f1.Header.FlowSequence, f2.Header.FlowSequence)
	}
}
