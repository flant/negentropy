package tools

import (
	"encoding/json"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

type VaultPayload struct {
	Data json.RawMessage `json:"data"`
}

func UnmarshalVaultResponse(b []byte) gjson.Result {
	payload := &VaultPayload{}
	err := json.Unmarshal(b, payload)
	Expect(err).ToNot(HaveOccurred())

	return gjson.Parse(string(payload.Data))
}
