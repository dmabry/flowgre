// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package barrage

import (
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
	// GenerateOptionsData creates an options data packet (IPFIX only).
	// Returns nil if the protocol does not support options templates.
	GenerateOptionsData(sourceID int, session *netflow.Session) []byte
	// GenerateData creates a data packet with the given number of flows.
	GenerateData(flowCount int, sourceID int, srcRange, dstRange string, session *netflow.Session) []byte
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

func (g netflowGenerator) GenerateOptionsData(sourceID int, session *netflow.Session) []byte {
	// NetFlow v9 does not support options templates
	return nil
}

func (g netflowGenerator) GenerateData(flowCount int, sourceID int, srcRange, dstRange string, session *netflow.Session) []byte {
	flow := netflow.GenerateDataNetflow(flowCount, sourceID, srcRange, dstRange, 0, session, g.profile)
	buf := flow.ToBytes()
	return buf.Bytes()
}

// ipfixGenerator implements FlowGenerator for IPFIX (RFC 7011).
type ipfixGenerator struct{}

func (g ipfixGenerator) Label() string { return "IPFIX Worker" }

func (g ipfixGenerator) GenerateTemplate(sourceID int, session *netflow.Session) []byte {
	tFlow := ipfix.GenerateTemplateIPFIX(sourceID, session)
	buf := tFlow.ToBytes()
	return buf.Bytes()
}

func (g ipfixGenerator) GenerateOptionsData(sourceID int, session *netflow.Session) []byte {
	oFlow := ipfix.GenerateOptionsDataIPFIX(sourceID, session)
	buf := oFlow.ToBytes()
	return buf.Bytes()
}

func (g ipfixGenerator) GenerateData(flowCount int, sourceID int, srcRange, dstRange string, session *netflow.Session) []byte {
	flow := ipfix.GenerateDataIPFIX(flowCount, sourceID, srcRange, dstRange, 0, session)
	buf := flow.ToBytes()
	return buf.Bytes()
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
func IPFIX() FlowGenerator { return ipfixGenerator{} }
