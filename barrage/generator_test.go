// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package barrage

import (
	"testing"

	"github.com/dmabry/flowgre/netflow"
)

func TestNetFlow_Profile_Default(t *testing.T) {
	t.Parallel()

	gen := NetFlow()
	_, ok := gen.(netflowGenerator)
	if !ok {
		t.Fatal("expected netflowGenerator type")
	}
}

func TestNetFlow_Profile_Minimal(t *testing.T) {
	t.Parallel()

	gen := NetFlow(&netflow.MinimalProfile{})
	ng, ok := gen.(netflowGenerator)
	if !ok {
		t.Fatal("expected netflowGenerator type")
	}

	session := netflow.NewSession()
	templateBytes := ng.GenerateTemplate(1, session)
	if len(templateBytes) == 0 {
		t.Fatal("expected non-empty template bytes")
	}

	// Minimal profile template should be smaller than generic (7 fields vs 18)
	genericGen := NetFlow(&netflow.GenericProfile{})
	genericNG := genericGen.(netflowGenerator)
	genericTemplateBytes := genericNG.GenerateTemplate(1, session)

	if len(templateBytes) >= len(genericTemplateBytes) {
		t.Errorf("minimal template (%d bytes) should be smaller than generic (%d bytes)",
			len(templateBytes), len(genericTemplateBytes))
	}
}

func TestNetFlow_Profile_Extended(t *testing.T) {
	t.Parallel()

	gen := NetFlow(&netflow.ExtendedProfile{})
	ng, ok := gen.(netflowGenerator)
	if !ok {
		t.Fatal("expected netflowGenerator type")
	}

	session := netflow.NewSession()
	templateBytes := ng.GenerateTemplate(1, session)
	if len(templateBytes) == 0 {
		t.Fatal("expected non-empty template bytes")
	}

	// Extended profile template should be larger than minimal
	minimalGen := NetFlow(&netflow.MinimalProfile{})
	minimalNG := minimalGen.(netflowGenerator)
	minimalTemplateBytes := minimalNG.GenerateTemplate(1, session)

	if len(templateBytes) <= len(minimalTemplateBytes) {
		t.Errorf("extended template (%d bytes) should be larger than minimal (%d bytes)",
			len(templateBytes), len(minimalTemplateBytes))
	}
}

func TestNetFlow_Profile_DataGeneration(t *testing.T) {
	t.Parallel()

	profiles := []struct {
		name    string
		profile netflow.FlowProfile
	}{
		{"generic", &netflow.GenericProfile{}},
		{"minimal", &netflow.MinimalProfile{}},
		{"extended", &netflow.ExtendedProfile{}},
	}

	for _, tc := range profiles {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gen := NetFlow(tc.profile)
			ng := gen.(netflowGenerator)

			session := netflow.NewSession()
			dataBytes, err := ng.GenerateData(5, 1, "10.0.0.0/8", "10.0.0.0/8", session)
			if err != nil {
				t.Fatalf("GenerateData error: %v", err)
			}

			if len(dataBytes) == 0 {
				t.Fatal("expected non-empty data bytes")
			}
		})
	}
}

func TestNetFlow_Profile_Label(t *testing.T) {
	t.Parallel()

	gen := NetFlow()
	if gen.Label() != "Worker" {
		t.Errorf("expected label 'Worker', got %q", gen.Label())
	}
}

func TestNetFlow_Profile_NoOptionsData(t *testing.T) {
	t.Parallel()

	gen := NetFlow()
	ng := gen.(netflowGenerator)

	session := netflow.NewSession()
	result := ng.GenerateOptionsData(1, session)
	if result != nil {
		t.Error("NetFlow v9 should not support options data")
	}
}
