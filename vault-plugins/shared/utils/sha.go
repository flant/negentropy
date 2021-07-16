package utils

import (
	"fmt"

	"crypto/sha256"
)

func ShaEncode(input string) string {
	hasher := sha256.New()

	hasher.Write([]byte(input))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
