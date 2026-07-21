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
func RandStringBytes(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		idx, err := CryptoRandomNumber(int64(len(letterBytes)))
		if err != nil {
			return "", fmt.Errorf("generate random char at index %d: %w", i, err)
		}
		b[i] = letterBytes[idx]
	}
	return string(b), nil
}

// CryptoRandomNumber generates a cryptographically secure random number in [0, max).
func CryptoRandomNumber(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, fmt.Errorf("crypto random failed: %w", err)
	}
	return n.Int64(), nil
}

// GenerateRand16 generates a random uint16 in [0, max).
func GenerateRand16(max int) (uint16, error) {
	n, err := CryptoRandomNumber(int64(max))
	if err != nil {
		return 0, fmt.Errorf("generate rand16: %w", err)
	}
	return uint16(n), nil
}

// GenerateRand32 generates a random uint32 in [0, max).
func GenerateRand32(max int) (uint32, error) {
	n, err := CryptoRandomNumber(int64(max))
	if err != nil {
		return 0, fmt.Errorf("generate rand32: %w", err)
	}
	return uint32(n), nil
}

// RandomNum generates a random integer in [min, max).
func RandomNum(min, max int) (int, error) {
	n, err := CryptoRandomNumber(int64(max - min))
	if err != nil {
		return 0, fmt.Errorf("generate random num in [%d, %d): %w", min, max, err)
	}
	return int(n) + min, nil
}