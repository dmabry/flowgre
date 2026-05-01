// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import "time"

// Session tracks per-invocation state for NetFlow generation.
// Replaces the previous package-level globals StartTime and flowSequence.
type Session struct {
	startTime    int64
	flowSequence uint32
}

// NewSession creates a fresh session with current time as start.
func NewSession() *Session {
	return &Session{
		startTime:    time.Now().UnixNano(),
		flowSequence: 0,
	}
}

// StartTime returns the session's start timestamp (nanoseconds since epoch).
func (s *Session) StartTime() int64 {
	return s.startTime
}

// NextSeq increments and returns the next flow sequence number.
func (s *Session) NextSeq() uint32 {
	s.flowSequence++
	return s.flowSequence
}
