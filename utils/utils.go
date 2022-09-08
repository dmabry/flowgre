// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Util funcs used throughout Flowgre

package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

// Constant used for generating random strings... not cryptographically safe.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandStringBytes Generates a random string of given length
func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// BinaryDecoder decodes the given payload from a binary stream and puts it in dest
func BinaryDecoder(payload io.Reader, dests ...interface{}) error {
	for _, dest := range dests {
		err := binary.Read(payload, binary.BigEndian, dest)
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateRand16 Generates random uint16 num within the given max
func GenerateRand16(max int) uint16 {
	return uint16(rand.Intn(max))
}

// IPto32 Converts given IP string to uint32 representation
func IPto32(s string) uint32 {
	ip := net.ParseIP(s)
	return binary.BigEndian.Uint32(ip.To4())
}

// GenerateRand32 Generates a random uint32 within the given max
func GenerateRand32(max int) uint32 {
	return uint32(rand.Intn(max))
}

// RandomNum Generates a random number between the given min and max
func RandomNum(min, max int) int {
	return rand.Intn(max-min) + min
}

// ToBytes Converts a given interface to a byte stream.
// Not used currently, but handy to have for later maybe.  Did not work for encoding Netflow packets as it
// encoded field names.
func ToBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RandomIP Picks a random IP from the given CIDR
// TODO: Better error handling needed
func RandomIP(cidr string) (net.IP, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Println("[ERROR] Parsing CIDR", cidr, " failed. error: ", err)
	}
	ipMin := ipNet.IP
	ipMax, _ := GetLastIP(ipNet)
	ipMinNum := IPToNum(ipMin)
	ipMaxNum := IPToNum(ipMax)
	rand.Seed(time.Now().UnixNano())
	randIPNum := uint32(rand.Int31n(int32(ipMaxNum-ipMinNum)) + int32(ipMinNum))
	randIP := NumToIP(randIPNum)
	//check if in range
	if ipNet.Contains(randIP) {
		return randIP, nil
	}
	return nil, errors.New("random IP broken")
}

// GetLastIP Gets the last IP of a given Network
func GetLastIP(ipNet *net.IPNet) (net.IP, error) {
	ip := make(net.IP, len(ipNet.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(ipNet.IP.To4())|^binary.BigEndian.Uint32(ipNet.Mask))
	return ip, nil
}

// IPToNum Converts given IP to uint32
func IPToNum(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// NumToIP Converts given uint32 to IP
func NumToIP(num uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, num)
	return ip
}

// SendPacket Takes a given byte stream and puts on the wire towards the given host
func SendPacket(conn *net.UDPConn, addr *net.UDPAddr, data bytes.Buffer) {
	n, err := conn.WriteTo(data.Bytes(), addr)
	if err != nil {
		log.Fatal("Write:", err)
	}
	fmt.Println("Sent", n, "bytes", conn.LocalAddr(), "->", addr)
}
