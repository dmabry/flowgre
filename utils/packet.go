// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package utils

import (
	"bytes"
	"fmt"
	"log"
	"net"
)

// SendPacket sends a byte buffer over UDP to the given address.
func SendPacket(conn *net.UDPConn, addr *net.UDPAddr, data bytes.Buffer, verbose bool) (int, error) {
	n, err := conn.WriteTo(data.Bytes(), addr)
	if err != nil {
		log.Println("Write:", err)
		return 0, err
	}
	if verbose {
		fmt.Println("Sent", n, "bytes", conn.LocalAddr(), "->", addr)
	}
	return n, err
}
