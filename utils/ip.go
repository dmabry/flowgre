// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package utils

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// IPto32 converts an IPv4 string to its uint32 representation.
func IPto32(s string) uint32 {
	ip := net.ParseIP(s)
	return binary.BigEndian.Uint32(ip.To4())
}

// RandomIP picks a random IP from the given CIDR range.
func RandomIP(cidr string) (net.IP, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("parsing CIDR %s: %w", cidr, err)
	}

	ipMin := ipNet.IP
	ipMax, _ := GetLastIP(ipNet)
	ipMinNum := IPToNum(ipMin)
	ipMaxNum := IPToNum(ipMax)

	var randIP net.IP
	if ipMinNum == ipMaxNum {
		// Only one IP in the range
		randIP = NumToIP(ipMinNum)
	} else {
		rangeSize := int64(ipMaxNum - ipMinNum)
		offset := CryptoRandomNumber(rangeSize)
		randIPNum := uint32(offset + int64(ipMinNum))
		randIP = NumToIP(randIPNum)
	}

	if ipNet.Contains(randIP) {
		return randIP, nil
	}
	return nil, errors.New("random IP out of range")
}

// GetLastIP returns the last IP address in the given network.
func GetLastIP(ipNet *net.IPNet) (net.IP, error) {
	ip := make(net.IP, len(ipNet.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(ipNet.IP.To4())|^binary.BigEndian.Uint32(ipNet.Mask))
	return ip, nil
}

// IPToNum converts an IPv4 address to its uint32 representation.
func IPToNum(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// NumToIP converts a uint32 to an IPv4 address.
func NumToIP(num uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, num)
	return ip
}
