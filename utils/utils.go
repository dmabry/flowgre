package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func BinaryDecoder(payload io.Reader, dests ...interface{}) error {
	for _, dest := range dests {
		err := binary.Read(payload, binary.BigEndian, dest)
		if err != nil {
			return err
		}
	}
	return nil
}
func GenRand16(max int) uint16 {
	return uint16(rand.Intn(max))
}

func IPto32(s string) uint32 {
	ip := net.ParseIP(s)
	return binary.BigEndian.Uint32(ip.To4())
}

func GenRand32(max int) uint32 {
	return uint32(rand.Intn(max))
}

func RandomNum(min, max int) int {
	return rand.Intn(max-min) + min
}

// Not used currently, but handy to have for later maybe
func ToBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func RandomIP(cidr string) (net.IP, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Println("[ERROR] Parsing CIDR", cidr, " failed. error: ", err)
	}
	ipMin := ipNet.IP
	ipMax, _ := getLastIP(ipNet)
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

	//fmt.Println("ipMin: ", ipMin.String(), " ipMax: ", ipMax.String())
}

func getLastIP(ipNet *net.IPNet) (net.IP, error) {
	ip := make(net.IP, len(ipNet.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(ipNet.IP.To4())|^binary.BigEndian.Uint32(ipNet.Mask))
	return ip, nil
}

func IPToNum(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func NumToIP(num uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, num)
	return ip
}
