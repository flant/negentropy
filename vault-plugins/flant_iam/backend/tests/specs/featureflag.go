package specs

import (
	"fmt"
	"math/rand"
	"net/url"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

func FeatureFlagCrudSpec(FeatureFlagAPI api.TestAPI) {
	_ = Describe("Feature Flag", func() {
		Describe("payload", func() {
			DescribeTable("identifier",
				func(name interface{}, statusCodeCondition string) {
					payload := fixtures.RandomFeatureFlagCreatePayload()
					payload["name"] = name

					params := api.Params{"expectStatus": api.ExpectStatus(statusCodeCondition)}

					FeatureFlagAPI.Create(params, nil, payload)
				},
				Entry("number allowed", rand.Intn(32), "%d == 201"), // the matter of fact ¯\_(ツ)_/¯
				Entry("absent identifier forbidden", nil, "%d >= 400"),
				Entry("empty string forbidden", "", "%d >= 400"),
				Entry("array forbidden", []string{"a"}, "%d >= 400"),
				Entry("object forbidden", map[string]int{"a": 1}, "%d >= 400"),
			)
		})

		It("can be created", func() {
			createPayload := fixtures.RandomFeatureFlagCreatePayload()

			params := api.Params{
				"expectPayload": func(json gjson.Result) {
					Expect(json.Get("feature_flag").Map()).To(HaveKey("name"))
				},
			}
			FeatureFlagAPI.Create(params, url.Values{}, createPayload)
		})

		It("can be listed", func() {
			createPayload := fixtures.RandomFeatureFlagCreatePayload()
			FeatureFlagAPI.Create(api.Params{}, url.Values{}, createPayload)
			FeatureFlagAPI.List(api.Params{}, url.Values{})
		})

		It("has identifying fields in list", func() {
			createPayload := fixtures.RandomFeatureFlagCreatePayload()
			FeatureFlagAPI.Create(api.Params{}, url.Values{}, createPayload)

			params := api.Params{
				"expectPayload": func(json gjson.Result) {
					Expect(json.Map()).To(HaveKey("names"))
					expectedFF := gjson.Parse(fmt.Sprintf("{\"archiving_hash\":0,\"archiving_timestamp\":0,\"name\":\"%s\"}", createPayload["name"]))
					CheckArrayContainsElement(json.Get("names").Array(), expectedFF)
				},
			}
			FeatureFlagAPI.List(params, url.Values{})
		})

		It("can be deleted", func() {
			createPayload := fixtures.RandomFeatureFlagCreatePayload()

			var createdData gjson.Result
			FeatureFlagAPI.Create(api.Params{
				"expectPayload": func(json gjson.Result) {
					createdData = json
				},
			}, nil, createPayload)

			FeatureFlagAPI.Delete(api.Params{
				"name": createdData.Get("feature_flag.name").String(),
			}, nil)

			FeatureFlagAPI.List(api.Params{
				"expectPayload": func(json gjson.Result) {
					Expect(json.Map()).To(HaveKey("names"))

					expectedName := gjson.Parse(fmt.Sprintf("%q", createdData.Get("name").String()))
					Expect(json.Get("names").Array()).ToNot(ContainElement(expectedName))
				},
			}, url.Values{})
		})

		It("when does not exist", func() {
			FeatureFlagAPI.Delete(api.Params{
				"name":         "not-exists",
				"expectStatus": api.ExpectExactStatus(404),
			}, nil)
		})
	})
}

func CheckArrayContainsElement(array []gjson.Result, element gjson.Result) {
	element.Map()
	var mapArray []map[string]gjson.Result
	for i := range array {
		mapArray = append(mapArray, array[i].Map())
	}
	Expect(mapArray).To(ContainElement(element.Map()))
}
