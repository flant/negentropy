package tools

import (
	"encoding/hex"
	"math/rand"
	"strconv"
	"time"
)

func RandomStr() string {
	rand.Seed(time.Now().UnixNano())

	entityName := make([]byte, 20)
	_, err := rand.Read(entityName)
	if err != nil {
		panic("not generate entity name")
	}

	return hex.EncodeToString(entityName)
}

func TimeStr() string {
	return strconv.Itoa(int(time.Now().UnixNano()))
}
