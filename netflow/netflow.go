// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Netflow v9 funcs and structs used for generating netflow packet to be put on the wire

package netflow

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
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
	header := new(Header).Generate(flowCount, sourceID, session)
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

// countDataRecordsRange computes the minimum and maximum possible record
// counts for a Data FlowSet given its data length and record size.
// The FlowSet data length includes record bytes plus 0-3 bytes of zero padding
// to reach a 32-bit boundary (RFC 3954 §9). Padding is recommended but not
// required (RFC 3954 §5.3), so multiple record counts may be valid.
// Returns (minCount, maxCount) or (-1, -1) if no valid padding exists.
func countDataRecordsRange(dataLen int, recordSize int) (int, int) {
	minCount := -1
	maxCount := -1

	for pad := 0; pad < 4; pad++ {
		dataBytes := dataLen - pad
		if dataBytes > 0 && dataBytes%recordSize == 0 {
			count := dataBytes / recordSize
			if minCount < 0 || count < minCount {
				minCount = count
			}
			if maxCount < 0 || count > maxCount {
				maxCount = count
			}
		}
	}

	return minCount, maxCount
}

// countDataRecords returns the single record count when padding is unambiguous,
// or -1 when multiple interpretations exist. Deprecated: prefer countDataRecordsRange.
func countDataRecords(dataLen int, recordSize int) int {
	min, max := countDataRecordsRange(dataLen, recordSize)
	if min == max {
		return min
	}
	return -1
}

// IsValidNetFlow validates the given payload as a structurally correct NetFlow v9 packet.
// It checks the fixed header, iterates over all FlowSets, and validates that each
// FlowSet has a valid ID, minimum length, and remains within packet boundaries.
func IsValidNetFlow(payload []byte, nfVersion int) (bool, error) {
	if len(payload) < 20 {
		return false, fmt.Errorf("payload too short for NetFlow v9 header: %d bytes", len(payload))
	}

	header := Header{}
	reader := bytes.NewReader(payload)
	err := binary.Read(reader, binary.BigEndian, &header)
	if err != nil {
		return false, fmt.Errorf("failed to parse NetFlow v9 header: %w", err)
	}
	if header.Version != uint16(nfVersion) {
		return false, fmt.Errorf("header version doesn't match: got %d, expected %d", header.Version, nfVersion)
	}

	// Validate FlowSets
	offset := 20
	limit := len(payload)
	setCount := 0

	// Track template record sizes for Data FlowSet record counting
	templateRecordSizes := make(map[uint16]int)

	// Deferred Data FlowSets (template not yet seen)
	type deferredDataInfo struct {
		fsOffset  int
		fsID      uint16
		fsDataLen int
	}
	var deferredData []deferredDataInfo

	// totalMinRecords and totalMaxRecords track the range of possible record
	// counts. Template and Options Template records are counted exactly. Data
	// FlowSet records use countDataRecordsRange which may return a range when
	// optional padding makes the count ambiguous (RFC 3954 §5.3, §9).
	totalMinRecords := 0
	totalMaxRecords := 0

	for offset < limit {
		if offset+4 > limit {
			return false, fmt.Errorf("insufficient data for FlowSet header at offset %d", offset)
		}

		setID := binary.BigEndian.Uint16(payload[offset : offset+2])
		setLength := binary.BigEndian.Uint16(payload[offset+2 : offset+4])

		// NetFlow v9: FlowSetID 0 = Template, 1 = Options Template, >=256 = Data
		// IDs 2-255 are reserved/unassigned
		if setID >= 2 && setID <= 255 {
			return false, fmt.Errorf("reserved FlowSet ID %d at offset %d", setID, offset)
		}

		if int(setLength) < 4 {
			return false, fmt.Errorf("FlowSet length %d is less than minimum 4 at offset %d", setLength, offset)
		}

		setEnd := offset + int(setLength)
		if setEnd > limit {
			return false, fmt.Errorf("FlowSet at offset %d extends beyond packet boundary (end %d > length %d)", offset, setEnd, limit)
		}

		// Validate set contents based on type
		remaining := int(setLength) - 4 // subtract FlowSet header
		setOffset := offset + 4

		switch setID {
		case 0:
			// Template FlowSet: one or more Template records
			recordsParsed := 0
			for remaining >= 4 {
				if setOffset+4 > limit {
					return false, fmt.Errorf("Template record header extends beyond packet at offset %d", setOffset)
				}
				tmplID := binary.BigEndian.Uint16(payload[setOffset : setOffset+2])
				fieldCount := binary.BigEndian.Uint16(payload[setOffset+2 : setOffset+4])
				if tmplID < 256 {
					return false, fmt.Errorf("Template ID %d is below 256 at offset %d", tmplID, setOffset)
				}
				// Each field specifier is 4 bytes (Type + Length)
				fieldSpecSize := int(fieldCount) * 4
				if 4+fieldSpecSize > remaining {
					return false, fmt.Errorf("Template record at offset %d exceeds FlowSet boundary", setOffset)
				}
				// Compute record size (sum of field lengths) and validate each field
				recordSize := 0
				for f := uint16(0); f < fieldCount; f++ {
					fOffset := setOffset + 4 + int(f)*4
					if fOffset+4 > limit {
						return false, fmt.Errorf("field specifier %d extends beyond packet at offset %d", f, fOffset)
					}
					fLength := binary.BigEndian.Uint16(payload[fOffset+2 : fOffset+4])
					if fLength == 0 {
						return false, fmt.Errorf("field %d has zero length", f)
					}
					recordSize += int(fLength)
				}
				templateRecordSizes[tmplID] = recordSize
				setOffset += 4 + fieldSpecSize
				remaining -= 4 + fieldSpecSize
				recordsParsed++
			}
			if recordsParsed == 0 {
				return false, fmt.Errorf("Template FlowSet at offset %d contains no records", offset)
			}
			totalMinRecords += recordsParsed
			totalMaxRecords += recordsParsed
		case 1:
			// Options Template FlowSet (RFC 3954 Section 6.1)
			// Per-record layout:
			//   Template ID (2 bytes) - must be >= 256
			//   Option Scope Length (2 bytes) - total bytes of scope field specifiers
			//   Option Length (2 bytes) - total bytes of option field specifiers
			//   Scope Field Specifiers (Option Scope Length bytes)
			//   Option Field Specifiers (Option Length bytes)
			recordsParsed := 0
			for remaining >= 6 {
				if setOffset+6 > limit {
					return false, fmt.Errorf("Options Template record header extends beyond packet at offset %d", setOffset)
				}
				tmplID := binary.BigEndian.Uint16(payload[setOffset : setOffset+2])
				scopeLen := binary.BigEndian.Uint16(payload[setOffset+2 : setOffset+4])
				optLen := binary.BigEndian.Uint16(payload[setOffset+4 : setOffset+6])

				if tmplID < 256 {
					return false, fmt.Errorf("Options Template ID %d is below 256 at offset %d", tmplID, setOffset)
				}
				// Scope and option lengths must be multiples of 4 (field specifier size)
				if scopeLen%4 != 0 {
					return false, fmt.Errorf("Options Template Scope Length %d is not a multiple of 4 at offset %d", scopeLen, setOffset)
				}
				if optLen%4 != 0 {
					return false, fmt.Errorf("Options Template Option Length %d is not a multiple of 4 at offset %d", optLen, setOffset)
				}

				optsRecordSize := 6 + int(scopeLen) + int(optLen)
				if optsRecordSize > remaining {
					return false, fmt.Errorf("Options Template record at offset %d exceeds FlowSet boundary", setOffset)
				}

				// Validate scope field specifiers and sum field lengths for record size
				optsDataSize := 0
				scopeFieldCount := int(scopeLen) / 4
				for s := 0; s < scopeFieldCount; s++ {
					fOffset := setOffset + 6 + s*4
					if fOffset+4 > limit {
						return false, fmt.Errorf("scope field specifier %d extends beyond packet at offset %d", s, fOffset)
					}
					fLength := binary.BigEndian.Uint16(payload[fOffset+2 : fOffset+4])
					if fLength == 0 {
						return false, fmt.Errorf("scope field %d has zero length", s)
					}
					optsDataSize += int(fLength)
				}

				// Validate option field specifiers and sum field lengths for record size
				optFieldCount := int(optLen) / 4
				for o := 0; o < optFieldCount; o++ {
					fOffset := setOffset + 6 + int(scopeLen) + o*4
					if fOffset+4 > limit {
						return false, fmt.Errorf("option field specifier %d extends beyond packet at offset %d", o, fOffset)
					}
					fLength := binary.BigEndian.Uint16(payload[fOffset+2 : fOffset+4])
					if fLength == 0 {
						return false, fmt.Errorf("option field %d has zero length", o)
					}
					optsDataSize += int(fLength)
				}

				templateRecordSizes[tmplID] = optsDataSize
				setOffset += optsRecordSize
				remaining -= optsRecordSize
				recordsParsed++
			}
			if recordsParsed == 0 {
				return false, fmt.Errorf("Options Template FlowSet at offset %d contains no records", offset)
			}
			totalMinRecords += recordsParsed
			totalMaxRecords += recordsParsed
		default:
			// Data FlowSet (ID >= 256): records match a Template
			// A Data FlowSet must contain at least one record; a record can be
			// as small as a single 1-byte field. RFC 3954 §5.3 recommends
			// 32-bit padding but does not require it.
			if remaining < 1 {
				return false, fmt.Errorf("Data FlowSet at offset %d is too small to contain records", offset)
			}
			// Use template record size to count actual records
			if rs, ok := templateRecordSizes[setID]; ok && rs > 0 {
				minCount, maxCount := countDataRecordsRange(remaining, rs)
				if minCount < 0 {
					return false, fmt.Errorf("Data FlowSet at offset %d has invalid record layout for template %d", offset, setID)
				}
				totalMinRecords += minCount
				totalMaxRecords += maxCount
			} else {
				// Template not yet seen; defer counting
				deferredData = append(deferredData, deferredDataInfo{
					fsOffset:  offset,
					fsID:      setID,
					fsDataLen: remaining,
				})
			}
		}

		// Validate padding for Template and Options Template FlowSets only
		if setID <= 1 && remaining > 0 {
			for i := 0; i < remaining; i++ {
				if payload[setOffset+i] != 0 {
					return false, fmt.Errorf("non-zero padding byte at offset %d in FlowSet %d", setOffset+i, offset)
				}
			}
		}

		setCount++
		offset = setEnd
	}

	// Process deferred Data FlowSets now that all templates are known.
	// Data-only packets (template sent in a prior packet) are accepted;
	// their records can't be counted without the template. Each unresolved
	// Data FlowSet must contain at least one record.
	hasUnresolvedData := false
	for _, info := range deferredData {
		rs, ok := templateRecordSizes[info.fsID]
		if !ok || rs == 0 {
			// Template not in this packet; count at least 1 record per FlowSet.
			// The upper bound is unknown, so we can't enforce it.
			hasUnresolvedData = true
			totalMinRecords++
			continue
		}
		minCount, maxCount := countDataRecordsRange(info.fsDataLen, rs)
		if minCount < 0 {
			return false, fmt.Errorf("Data FlowSet at offset %d has invalid record layout for template %d", info.fsOffset, info.fsID)
		}
		totalMinRecords += minCount
		totalMaxRecords += maxCount
	}

	if setCount == 0 {
		return false, fmt.Errorf("NetFlow v9 packet contains no FlowSets")
	}

	// Validate header FlowCount against the range of possible record counts.
	// When padding makes the count ambiguous, the header resolves which
	// interpretation is correct (RFC 3954 §9).
	if header.FlowCount == 0 {
		return false, fmt.Errorf("header FlowCount is zero")
	}
	if totalMinRecords > math.MaxUint16 {
		return false, fmt.Errorf("record count %d exceeds FlowCount field capacity", totalMinRecords)
	}
	// When templates are missing, the upper bound is unknown; only enforce
	// the lower bound. When all templates are known, enforce the full range.
	if hasUnresolvedData {
		if int(header.FlowCount) < totalMinRecords {
			return false, fmt.Errorf("header FlowCount %d is less than minimum record count %d", header.FlowCount, totalMinRecords)
		}
	} else {
		if int(header.FlowCount) < totalMinRecords || int(header.FlowCount) > totalMaxRecords {
			return false, fmt.Errorf("header FlowCount %d is outside valid range [%d, %d]", header.FlowCount, totalMinRecords, totalMaxRecords)
		}
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
