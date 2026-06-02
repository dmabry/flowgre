// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package utils provides general-purpose utility functions for flowgre.
// Random number generation, IP math, and packet sending have been extracted
// to dedicated sub-packages (rand.go, ip.go, packet.go).
package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io"
)

// BinaryDecoder decodes the given payload from a binary stream into multiple destinations.
func BinaryDecoder(payload io.Reader, dests ...any) error {
	for _, dest := range dests {
		err := binary.Read(payload, binary.BigEndian, dest)
		if err != nil {
			return err
		}
	}
	return nil
}

// ToBytes converts an interface to a gob-encoded byte stream.
func ToBytes(key any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
