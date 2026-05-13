// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

// Netflow complete record
type Netflow struct {
	Header           Header
	TemplateFlowSets []TemplateFlowSet
	DataFlowSets     []DataFlowSet
}

// ToBytes Converts Netflow struct to a bytes buffer than can be written to the wire
func (n *Netflow) ToBytes() bytes.Buffer {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, &n.Header)
	if err != nil {
		log.Println("[ERROR] Issue writing header: ", err)
	}
	// Write Template flow if any exists
	if len(n.TemplateFlowSets) > 0 {
		for _, tFlow := range n.TemplateFlowSets {
			// Order FlowSetID, Length, Template(s)
			err := binary.Write(&buf, binary.BigEndian, tFlow.FlowSetID)
			if err != nil {
				log.Println("[ERROR] Issue writing Template FlowSetID: ", err)
			}
			err = binary.Write(&buf, binary.BigEndian, tFlow.Length)
			if err != nil {
				log.Println("[ERROR] Issue writing Template Length: ", err)
			}
			for _, template := range tFlow.Templates {
				// Order TemplateId, Field Count, Field(s)
				err = binary.Write(&buf, binary.BigEndian, template.TemplateID)
				if err != nil {
					log.Println("[ERROR] Issue writing Template ID: ", err)
				}
				err = binary.Write(&buf, binary.BigEndian, template.FieldCount)
				if err != nil {
					log.Println("[ERROR] Issue writing Template FieldCount: ", err)
				}
				for _, field := range template.Fields {
					err = binary.Write(&buf, binary.BigEndian, field.Type)
					if err != nil {
						log.Println("[ERROR} Issue writing Field Type: ", err)
					}
					err = binary.Write(&buf, binary.BigEndian, field.Length)
					if err != nil {
						log.Println("[ERROR} Issue writing Field Length: ", err)
					}
				}
			}
			// Padding to 32 bit boundary per Netflow v9 RFC
			if tFlow.Padding > 0 {
				padBytes := bytes.Repeat([]byte{0}, tFlow.Padding)
				err = binary.Write(&buf, binary.BigEndian, padBytes)
				if err != nil {
					log.Println("[ERROR] Issue writing Template Padding: ", err)
				}
			}
		}
	}
	// Write Data flow(s) if any exists
	if len(n.DataFlowSets) > 0 {
		for _, dFlow := range n.DataFlowSets {
			// Order FlowSetID, Length, Record(s)
			err := binary.Write(&buf, binary.BigEndian, dFlow.FlowSetID)
			if err != nil {
				log.Println("[ERROR] Issue writing Data FlowSetID: ", err)
			}
			err = binary.Write(&buf, binary.BigEndian, dFlow.Length)
			if err != nil {
				log.Println("[ERROR] Issue writing Data FlowSet Length: ", err)
			}
			for _, item := range dFlow.Items {
				err = binary.Write(&buf, binary.BigEndian, item)
				if err != nil {
					log.Println("[ERROR] Issue writing Data FlowSet Field: ", err)
				}
			}
			// Padding to 32 bit boundary per Netflow v9 RFC
			if dFlow.Padding != 0 {
				padtext := bytes.Repeat([]byte{byte(0)}, dFlow.Padding)
				err = binary.Write(&buf, binary.BigEndian, padtext)
				if err != nil {
					log.Println("[ERROR] Issue writing Data Padding: ", err)
				}
			}
		}
	}
	return buf
}

// GetNetFlowSizes Gets the size of a given Netflow and returns it as a String
func GetNetFlowSizes(netFlow Netflow) string {
	output := fmt.Sprintf("Header Size: %d bytes\n", netFlow.Header.size())
	tSize := 0
	for _, tFlow := range netFlow.TemplateFlowSets {
		tSize += tFlow.size()
	}
	output += fmt.Sprintf("Template Size: %d bytes\n", tSize)
	dSize := 0
	for _, dFlow := range netFlow.DataFlowSets {
		dSize += dFlow.size()
	}
	output += fmt.Sprintf("Data Size: %d bytes\n", dSize)
	return output
}
