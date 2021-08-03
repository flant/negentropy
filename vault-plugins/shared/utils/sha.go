package utils

import (
	"crypto/sha256"
	"fmt"
)

func ShaEncode(input string) string {
	hasher := sha256.New()

	hasher.Write([]byte(input))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
