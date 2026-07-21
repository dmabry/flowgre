// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package utils

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// IPto32 converts an IPv4 string to its uint32 representation.
// Handles nil, IPv6 (via IPv4-mapped prefix), and valid IPv4 gracefully without panicking.
// Note: Returns 0 for both invalid input and the valid address "0.0.0.0".
// Use ParseIPv4ToNum when error distinction is needed.
func IPto32(s string) uint32 {
	ip := net.ParseIP(s)
	if ip == nil {
		return 0
	}
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16]) // IPv4-mapped IPv6
	}
	if len(ip) == 4 {
		return binary.BigEndian.Uint32(ip)
	}
	return 0
}

// ParseIPv4ToNum converts an IPv4 string to its uint32 representation,
// returning an error for IPv6 or invalid input.
func ParseIPv4ToNum(s string) (uint32, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP address: %s", s)
	}
	if len(ip) == 16 {
		ipv4 := ip.To4()
		if ipv4 != nil {
			return binary.BigEndian.Uint32(ipv4), nil
		}
		return 0, fmt.Errorf("pure IPv6 not supported: %s", s)
	}
	if len(ip) == 4 {
		return binary.BigEndian.Uint32(ip), nil
	}
	return 0, fmt.Errorf("unrecognized IP format: %s", s)
}

// RandomIP picks a random IP from the given CIDR range.
func RandomIP(cidr string) (net.IP, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("parsing CIDR %s: %w", cidr, err)
	}

	ipMin := ipNet.IP
	ipMax := GetLastIP(ipNet)
	ipMinNum := IPToNum(ipMin)
	ipMaxNum := IPToNum(ipMax)

	var randIP net.IP
	if ipMinNum == ipMaxNum {
		// Only one IP in the range
		randIP = NumToIP(ipMinNum)
	} else {
		rangeSize := int64(ipMaxNum - ipMinNum)
		offset, err := CryptoRandomNumber(rangeSize)
		if err != nil {
			return nil, fmt.Errorf("generate random IP offset: %w", err)
		}
		randIPNum := uint32(offset + int64(ipMinNum))
		randIP = NumToIP(randIPNum)
	}

	if ipNet.Contains(randIP) {
		return randIP, nil
	}
	return nil, errors.New("random IP out of range")
}

// GetLastIP returns the last IP address in the given network.
func GetLastIP(ipNet *net.IPNet) net.IP {
	ip := make(net.IP, len(ipNet.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(ipNet.IP.To4())|^binary.BigEndian.Uint32(ipNet.Mask))
	return ip
}

// IPToNum converts an IPv4 address to its uint32 representation.
func IPToNum(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
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

// IsIPv6CIDR detects whether a CIDR string represents an IPv6 network.
func IsIPv6CIDR(cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ipNet.IP.To4() == nil
}

// RandomIPv6 generates a random IPv6 address within the given CIDR range.
func RandomIPv6(cidr string) (net.IP, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("parsing CIDR %s: %w", cidr, err)
	}
	if ipNet.IP.To4() != nil {
		return nil, fmt.Errorf("CIDR %s is IPv4, use RandomIP instead", cidr)
	}

	ip := make(net.IP, 16)
	copy(ip, ipNet.IP)

	ones, bits := ipNet.Mask.Size()
	hostBits := bits - ones

	if hostBits == 0 {
		// /128 — single address
		return ip, nil
	}

	// Generate random bytes for host portion
	hostBytes := make([]byte, 16)
	_, err = rand.Read(hostBytes)
	if err != nil {
		return nil, fmt.Errorf("generating random bytes: %w", err)
	}

	// Clear network bits, keep only host bits
	for i := 0; i < 16; i++ {
		byteBit := i * 8
		if byteBit+8 <= ones {
			// Entire byte is network bits — keep as-is
			continue
		} else if byteBit >= ones {
			// Entire byte is host bits
			ip[i] = hostBytes[i]
		} else {
			// Partial byte — boundary between network and host
			hostBitInByte := 8 - (ones - byteBit)
			mask := byte(0)
			for b := 0; b < hostBitInByte; b++ {
				mask |= byte(1) << b
			}
			ip[i] = ipNet.IP[i] | (hostBytes[i] & mask)
		}
	}

	if ipNet.Contains(ip) {
		return ip, nil
	}
	return nil, errors.New("random IPv6 out of range")
}

// GetLastIPv6 returns the last (broadcast-equivalent) IPv6 address in the network.
func GetLastIPv6(ipNet *net.IPNet) net.IP {
	ip := make(net.IP, 16)
	copy(ip, ipNet.IP)

	ones, _ := ipNet.Mask.Size()

	for i := 0; i < 16; i++ {
		byteBit := i * 8
		if byteBit+8 <= ones {
			// Entire byte is network bits — keep as-is
			continue
		} else if byteBit >= ones {
			// Entire byte is host bits — set to 1
			ip[i] = 0xff
		} else {
			// Partial byte — boundary between network and host
			hostBitInByte := 8 - (ones - byteBit)
			mask := byte(0)
			for b := 0; b < hostBitInByte; b++ {
				mask |= byte(1) << b
			}
			ip[i] = ipNet.IP[i] | mask
		}
	}
	return ip
}

// RandomIPCIDR is a unified dispatcher that auto-detects IPv4 vs IPv6
// and calls the appropriate random IP generation function.
func RandomIPCIDR(cidr string) (net.IP, error) {
	if IsIPv6CIDR(cidr) {
		return RandomIPv6(cidr)
	}
	return RandomIP(cidr)
}
