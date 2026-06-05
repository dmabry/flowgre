// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandStringBytes generates a random string of given length using crypto/rand.
func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[CryptoRandomNumber(int64(len(letterBytes)))]
	}
	return string(b)
}

// CryptoRandomNumber generates a cryptographically secure random number in [0, max).
func CryptoRandomNumber(max int64) int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		panic(fmt.Errorf("crypto random failed: %w", err))
	}
	return n.Int64()
}

// GenerateRand16 generates a random uint16 in [0, max).
func GenerateRand16(max int) uint16 {
	return uint16(CryptoRandomNumber(int64(max)))
}

// GenerateRand32 generates a random uint32 in [0, max).
func GenerateRand32(max int) uint32 {
	return uint32(CryptoRandomNumber(int64(max)))
}

// RandomNum generates a random integer in [min, max).
func RandomNum(min, max int) int {
	return int(CryptoRandomNumber(int64(max-min))) + min
}
