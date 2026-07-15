// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package barrage

import (
	"fmt"

	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/netflow"
)

// FlowGenerator abstracts protocol-specific packet generation so that
// the barrage worker loop is shared between NetFlow and IPFIX.
type FlowGenerator interface {
	// Label returns the human-readable protocol name used in log messages.
	Label() string
	// GenerateTemplate creates the initial template packet for a source ID.
	GenerateTemplate(sourceID int, session *netflow.Session) []byte
	// GenerateTemplateWithSeq creates a template packet with the current
	// sequence number. Used for template retransmissions.
	GenerateTemplateWithSeq(sourceID int, session *netflow.Session) []byte
	// GenerateOptionsData creates an options data packet (IPFIX only).
	// Returns nil if the protocol does not support options templates.
	GenerateOptionsData(sourceID int, session *netflow.Session) []byte
	// GenerateData creates a data packet with the given number of flows.
	GenerateData(flowCount int, sourceID int, srcRange, dstRange string, session *netflow.Session) ([]byte, error)
	// ForWorker returns a per-worker copy with its own sequence counter.
	// Each worker must have an independent sequence per RFC 7011 §3.1.
	ForWorker() FlowGenerator
}

// netflowGenerator implements FlowGenerator for NetFlow v9.
type netflowGenerator struct {
	profile netflow.FlowProfile
}

func (g netflowGenerator) Label() string { return "Worker" }

func (g netflowGenerator) GenerateTemplate(sourceID int, session *netflow.Session) []byte {
	tFlow := netflow.GenerateTemplateNetflow(sourceID, session, g.profile)
	buf := tFlow.ToBytes()
	return buf.Bytes()
}

func (g netflowGenerator) GenerateTemplateWithSeq(sourceID int, session *netflow.Session) []byte {
	// NetFlow v9 doesn't have sequence numbers in the same way
	return g.GenerateTemplate(sourceID, session)
}

func (g netflowGenerator) GenerateOptionsData(sourceID int, session *netflow.Session) []byte {
	// NetFlow v9 does not support options templates
	return nil
}

func (g netflowGenerator) GenerateData(flowCount int, sourceID int, srcRange, dstRange string, session *netflow.Session) ([]byte, error) {
	flow, err := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session, g.profile)
	if err != nil {
		return nil, fmt.Errorf("GenerateDataNetflow failed: %w", err)
	}
	buf := flow.ToBytes()
	return buf.Bytes(), nil
}

// ForWorker returns the same generator (NetFlow uses session-based sequencing).
func (g netflowGenerator) ForWorker() FlowGenerator { return g }

// ipfixGenerator implements FlowGenerator for IPFIX (RFC 7011).
type ipfixGenerator struct {
	seq *ipfix.IPFIXSequence
}

func (g ipfixGenerator) Label() string { return "IPFIX Worker" }

func (g ipfixGenerator) GenerateTemplate(sourceID int, session *netflow.Session) []byte {
	tFlow := ipfix.GenerateTemplateIPFIX(sourceID, g.seq)
	buf, _ := tFlow.ToBytes()
	return buf.Bytes()
}

func (g ipfixGenerator) GenerateTemplateWithSeq(sourceID int, session *netflow.Session) []byte {
	// Regenerate template with current sequence number and export time
	tFlow := ipfix.GenerateTemplateIPFIX(sourceID, g.seq)
	buf, _ := tFlow.ToBytes()
	return buf.Bytes()
}

func (g ipfixGenerator) GenerateOptionsData(sourceID int, session *netflow.Session) []byte {
	oFlow := ipfix.GenerateOptionsDataIPFIX(sourceID, g.seq)
	buf, _ := oFlow.ToBytes()
	return buf.Bytes()
}

func (g ipfixGenerator) GenerateData(flowCount int, sourceID int, srcRange, dstRange string, session *netflow.Session) ([]byte, error) {
	flow, err := ipfix.GenerateDataIPFIX(flowCount, sourceID, srcRange, dstRange, 0, g.seq)
	if err != nil {
		return nil, fmt.Errorf("GenerateDataIPFIX failed: %w", err)
	}
	buf, err := flow.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("IPFIX ToBytes failed: %w", err)
	}
	return buf.Bytes(), nil
}

// ForWorker returns a new generator with its own IPFIXSequence.
// Each worker must have an independent sequence per RFC 7011 §3.1.
func (g ipfixGenerator) ForWorker() FlowGenerator {
	return ipfixGenerator{seq: ipfix.NewIPFIXSequence()}
}

// NetFlow returns a FlowGenerator for NetFlow v9.
// Optionally accepts a FlowProfile; defaults to GenericProfile.
func NetFlow(profile ...netflow.FlowProfile) FlowGenerator {
	p := netflow.FlowProfile(&netflow.GenericProfile{})
	if len(profile) > 0 && profile[0] != nil {
		p = profile[0]
	}
	return netflowGenerator{profile: p}
}

// IPFIX returns a FlowGenerator for IPFIX (RFC 7011).
func IPFIX() FlowGenerator {
	return ipfixGenerator{seq: ipfix.NewIPFIXSequence()}
}
