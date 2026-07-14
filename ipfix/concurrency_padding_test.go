// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package ipfix

import (
	"sync"
	"testing"

	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

// ---------------------------------------------------------------------------
// Padding alignment tests
// ---------------------------------------------------------------------------

func TestTemplateFlowSet_Padding_Alignment(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()

	cases := []struct {
		name    string
		profile IPFIXFlowProfile
	}{
		{"minimal", &MinimalIPFIXProfile{}},
		{"extended", &ExtendedIPFIXProfile{}},
		{"generic", &GenericIPFIXProfile{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tfs := new(TemplateFlowSet).Generate(session, tc.profile)
			// Total length should be aligned to 4-byte boundary per RFC 7011
			if tfs.Length%4 != 0 {
				t.Errorf("%s: TemplateFlowSet length %d should be 4-byte aligned",
					tc.name, tfs.Length)
			}
		})
	}
}

func TestOptionsTemplateFlowSet_Padding_Alignment(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	otfs := new(OptionsTemplateFlowSet).Generate(session)

	// Total length should be aligned to 4-byte boundary per RFC 7011
	if otfs.Length%4 != 0 {
		t.Errorf("OptionsTemplateFlowSet length %d should be 4-byte aligned", otfs.Length)
	}
}

func TestOptionsDataFlowSet_Padding_Alignment(t *testing.T) {
	t.Parallel()
	ods := new(OptionsDataFlowSet).Generate(618, "flowgre", 12345)

	// Total length should be aligned to 4-byte boundary per RFC 7011
	if ods.Length%4 != 0 {
		t.Errorf("OptionsDataFlowSet length %d should be 4-byte aligned", ods.Length)
	}
}

func TestDataFlowSet_Padding_Alignment(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()

	cases := []struct {
		name    string
		profile IPFIXFlowProfile
	}{
		{"minimal", &MinimalIPFIXProfile{}},
		{"extended", &ExtendedIPFIXProfile{}},
		{"generic", &GenericIPFIXProfile{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dfs, err := new(DataFlowSet).Generate(1, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session, tc.profile)
			if err != nil {
				t.Fatal(err)
			}
			// Total length should be aligned to 4-byte boundary per RFC 7011
			if dfs.Length%4 != 0 {
				t.Errorf("%s: DataFlowSet length %d should be 4-byte aligned",
					tc.name, dfs.Length)
			}
		})
	}
}

func TestPadding_VariesByProfile(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()

	// Different profiles have different field counts and sizes,
	// so padding requirements vary. Verify each aligns independently.
	type result struct {
		length  uint16
		padding int
	}

	results := make(map[string]result)

	for _, profile := range []IPFIXFlowProfile{
		&MinimalIPFIXProfile{},
		&ExtendedIPFIXProfile{},
		&GenericIPFIXProfile{},
	} {
		tfs := new(TemplateFlowSet).Generate(session, profile)
		results[profile.Name()] = result{length: tfs.Length, padding: tfs.Padding}
	}

	// Each profile should produce 4-byte-aligned length
	for name, r := range results {
		if r.length%4 != 0 {
			t.Errorf("%s: length %d not 4-byte aligned", name, r.length)
		}
		t.Logf("%s: length=%d, padding=%d", name, r.length, r.padding)
	}
}

// ---------------------------------------------------------------------------
// Concurrent safety tests
// ---------------------------------------------------------------------------

func TestSession_NextSeq_Concurrent(t *testing.T) {
	t.Parallel()
	s := netflow.NewSession()
	const numGoroutines = 100
	var wg sync.WaitGroup
	seen := make(map[uint32]bool)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seq := s.NextSeq()
			mu.Lock()
			if seen[seq] {
				t.Errorf("duplicate sequence number: %d", seq)
			}
			seen[seq] = true
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(seen) != numGoroutines {
		t.Errorf("expected %d unique sequences, got %d", numGoroutines, len(seen))
	}
}

func TestGenerateIPFIX_Concurrent_Safe(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	const numGoroutines = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	sequences := make([]uint32, 0, numGoroutines)
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ipfix, err := GenerateIPFIX(1, 618, "10.0.0.0/8", "10.0.0.0/8", session)
			if err != nil {
				errCh <- err
				return
			}

			mu.Lock()
			sequences = append(sequences, ipfix.Header.FlowSequence)
			mu.Unlock()
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatal(err)
	}

	// Verify all sequences are unique
	seen := make(map[uint32]bool)
	for _, seq := range sequences {
		if seen[seq] {
			t.Errorf("duplicate sequence number from concurrent generation: %d", seq)
		}
		seen[seq] = true
	}
}

func TestGenerateTemplateIPFIX_Concurrent_Safe(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	const numGoroutines = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	sequences := make([]uint32, 0, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ipfix := GenerateTemplateIPFIX(618, session)

			mu.Lock()
			sequences = append(sequences, ipfix.Header.FlowSequence)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Verify all sequences are unique
	seen := make(map[uint32]bool)
	for _, seq := range sequences {
		if seen[seq] {
			t.Errorf("duplicate sequence number from concurrent template generation: %d", seq)
		}
		seen[seq] = true
	}
}

// ---------------------------------------------------------------------------
// Profile field data alignment
// ---------------------------------------------------------------------------

func TestProfileFieldDataAlignment_Minimal(t *testing.T) {
	t.Parallel()
	profile := &MinimalIPFIXProfile{}
	fields := profile.TemplateFields()

	// Verify field count matches what MinimalIPFIXFlow expects
	if len(fields) != 7 {
		t.Errorf("MinimalIPFIXProfile: expected 7 fields, got %d", len(fields))
	}

	// Verify field types match MinimalIPFIXFlow struct
	expectedTypes := []uint16{
		InOctets, InPackets,
		SourceIPv4Address, DestinationIPv4Address,
		SourceTransportPort, DestinationTransportPort,
		ProtocolIdentifier,
	}
	for i, want := range expectedTypes {
		if fields[i].Type != want {
			t.Errorf("field[%d] type: got %d, want %d", i, fields[i].Type, want)
		}
	}
}

func TestProfileFieldDataAlignment_Extended(t *testing.T) {
	t.Parallel()
	profile := &ExtendedIPFIXProfile{}
	fields := profile.TemplateFields()

	// Verify field count matches ExtendedIPFIXProfile definition
	if len(fields) != 17 {
		t.Errorf("ExtendedIPFIXProfile: expected 17 fields, got %d", len(fields))
	}

	// Verify field types match ExtendedIPFIXProfile struct
	expectedTypes := []uint16{
		InOctets, OutOctets,
		InPackets, OutPackets,
		SourceIPv4Address, DestinationIPv4Address,
		SourceTransportPort, DestinationTransportPort,
		ProtocolIdentifier,
		TCPFlags,
		FlowStartMilliseconds, FlowEndMilliseconds,
		FlowDirection,
		IPClassOfService,
		FlowEndReason,
		SourceIPv6Address, DestinationIPv6Address,
	}
	for i, want := range expectedTypes {
		if fields[i].Type != want {
			t.Errorf("field[%d] type: got %d, want %d", i, fields[i].Type, want)
		}
	}
}

func TestProfileFieldDataAlignment_Generic(t *testing.T) {
	t.Parallel()
	profile := &GenericIPFIXProfile{}
	fields := profile.TemplateFields()

	// Verify field count matches GenericFlow struct
	if len(fields) != 19 {
		t.Errorf("GenericIPFIXProfile: expected 19 fields, got %d", len(fields))
	}

	// Verify first and last fields match expectations
	if fields[0].Type != InOctets {
		t.Errorf("first field type: got %d, want %d", fields[0].Type, InOctets)
	}
	if fields[len(fields)-1].Type != FlowEndReason {
		t.Errorf("last field type: got %d, want %d", fields[len(fields)-1].Type, FlowEndReason)
	}
}

// ---------------------------------------------------------------------------
// Padding edge cases
// ---------------------------------------------------------------------------

func TestDataFlowSet_Padding_ZeroFlowCount(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()

	dfs, err := new(DataFlowSet).Generate(0, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}

	// Even with zero flows, length should be 4-byte aligned
	if dfs.Length%4 != 0 {
		t.Errorf("DataFlowSet length %d not 4-byte aligned with zero flows", dfs.Length)
	}
}

func TestOptionsDataFlowSet_Padding_ProcessName(t *testing.T) {
	t.Parallel()

	// Test with various process name lengths to exercise padding calculations
	testCases := []struct {
		name string
		len  int
	}{
		{"short", 5},
		{"medium", 20},
		{"long", 50},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			processName := make([]byte, tc.len)
			for i := range processName {
				processName[i] = byte('a' + i%26)
			}
			ods := new(OptionsDataFlowSet).Generate(618, string(processName), 12345)
			if ods.Length%4 != 0 {
				t.Errorf("OptionsDataFlowSet length %d not 4-byte aligned with %d-char process name",
					ods.Length, tc.len)
			}
		})
	}
}
