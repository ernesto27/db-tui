// Package utils provides shared utility functions.
package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const randomAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var randomAlphabetLength = big.NewInt(int64(len(randomAlphabet)))

// RandomString returns a cryptographically random alphanumeric string of length.
func RandomString(length int) (string, error) {
	if length < 0 {
		return "", fmt.Errorf("random string length must not be negative")
	}

	value := make([]byte, length)
	for index := range value {
		randomIndex, err := rand.Int(rand.Reader, randomAlphabetLength)
		if err != nil {
			return "", fmt.Errorf("generate random character: %w", err)
		}
		value[index] = randomAlphabet[randomIndex.Int64()]
	}
	return string(value), nil
}
