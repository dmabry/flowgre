// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import "testing"

func FuzzIsValidNetFlow(f *testing.F) {
	// Add seed corpus with known valid packets
	session := NewSession()

	// Template-only packet
	tmplFlow := GenerateTemplateNetflow(618, session)
	tmplBuf := tmplFlow.ToBytes()
	f.Add(tmplBuf.Bytes())

	// Template + data packet
	nf, err := GenerateNetflow(5, 618, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		f.Fatal(err)
	}
	nfBuf := nf.ToBytes()
	f.Add(nfBuf.Bytes())

	// Data-only packet
	dataFlow, err := GenerateDataNetflow(3, 618, "10.0.0.0/8", "10.0.0.0/8", 443, session)
	if err != nil {
		f.Fatal(err)
	}
	dataBuf := dataFlow.ToBytes()
	f.Add(dataBuf.Bytes())

	// Minimal valid packet: header + template with 1 field
	f.Add([]byte{
		0x00, 0x09, // version 9
		0x00, 0x01, // flowCount 1
		0x00, 0x00, 0x03, 0xE8, // SysUptime 1000
		0x00, 0x00, 0x00, 0x00, // UnixSec 0
		0x00, 0x00, 0x00, 0x01, // FlowSequence 1
		0x00, 0x00, 0x02, 0x6A, // SourceID 618
		// Template FlowSet (ID=0), length=12 (4 header + 2 tmplID + 2 fieldCount + 4 field)
		0x00, 0x00, // FlowSetID 0
		0x00, 0x0C, // Length 12
		0x01, 0x00, // TemplateID 256
		0x00, 0x01, // FieldCount 1
		0x00, 0x01, // Field type 1
		0x00, 0x04, // Field length 4
	})

	// Short payload (too short for header)
	f.Add([]byte{0x00, 0x09, 0x00})

	// Empty payload
	f.Add([]byte{})

	// Random-looking bytes
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		// IsValidNetFlow should never panic on any input
		IsValidNetFlow(data, 9)
	})
}
