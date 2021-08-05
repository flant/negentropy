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
	if len(b) > 0 {
		err := json.Unmarshal(b, payload)
		Expect(err).ToNot(HaveOccurred())
	}

	return gjson.Parse(string(payload.Data))
}

func ToMap(v interface{}) map[string]interface{} {
	js, err := json.Marshal(v)
	Expect(err).ToNot(HaveOccurred())
	out := map[string]interface{}{}
	err = json.Unmarshal(js, &out)
	Expect(err).ToNot(HaveOccurred())

	return out
}
