package fixtures

import (
	"encoding/hex"
	"math/rand"
	"time"
)

func RandomStr() string {
	rand.Seed(time.Now().UnixNano())

	entityName := make([]byte, 20)
	rand.Read(entityName) // nolint:errcheck
	return hex.EncodeToString(entityName)
}
