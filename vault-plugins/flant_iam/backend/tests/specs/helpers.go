package specs

import (
	"encoding/json"

	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"

	. "github.com/onsi/gomega"
)

func IsSubsetExceptKeys(subset gjson.Result, set gjson.Result, keys ...string) {
	setMap := set.Map()
	subsetMap := subset.Map()
	for _, key := range keys {
		subsetMap[key] = setMap[key]
	}
	Expect(setMap).To(Equal(subsetMap))
}

func CheckArrayContainsElement(array []gjson.Result, element gjson.Result) {
	element.Map()
	var mapArray []map[string]gjson.Result
	for i := range array {
		mapArray = append(mapArray, array[i].Map())
	}
	Expect(mapArray).To(ContainElement(element.Map()))
}

func CreateRandomTenant(tenantsAPI api.TestAPI) model.Tenant {
	createPayload := fixtures.RandomTenantCreatePayload()

	var createdData gjson.Result
	tenantsAPI.Create(api.Params{
		"expectPayload": func(json gjson.Result) {
			createdData = json
		},
	}, nil, createPayload)
	rawTenant := createdData.Get("tenant")
	data := []byte(rawTenant.String())
	var tenant model.Tenant
	err := json.Unmarshal(data, tenant)
	Expect(err).ToNot(HaveOccurred())
	return tenant
}
