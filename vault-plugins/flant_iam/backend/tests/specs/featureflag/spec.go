package featureflag

import (
	"fmt"
	"math/rand"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
)

var TestAPI api.TestAPI

var _ = Describe("Feature Flag", func() {
	Describe("payload", func() {
		DescribeTable("identifier",
			func(name interface{}, statusCodeCondition string) {
				payload := fixtures.RandomFeatureFlagCreatePayload()
				payload["name"] = name

				params := api.Params{"expectStatus": api.ExpectStatus(statusCodeCondition)}

				TestAPI.Create(params, nil, payload)
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
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be listed", func() {
		createRandomFeatureFlag(TestAPI)
		TestAPI.List(api.Params{}, url.Values{})
	})

	It("has identifying fields in list", func() {
		createPayload := fixtures.RandomFeatureFlagCreatePayload()
		TestAPI.Create(api.Params{}, url.Values{}, createPayload)

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				Expect(json.Map()).To(HaveKey("names"))
				expectedFF := gjson.Parse(fmt.Sprintf("{\"archiving_hash\":0,\"archiving_timestamp\":0,\"name\":\"%s\"}", createPayload["name"]))
				specs.CheckArrayContainsElement(json.Get("names").Array(), expectedFF)
			},
		}
		TestAPI.List(params, url.Values{})
	})

	It("can be deleted", func() {
		createdFeatureFlagName := createRandomFeatureFlag(TestAPI)

		TestAPI.Delete(api.Params{
			"name": createdFeatureFlagName,
		}, nil)

		TestAPI.List(api.Params{
			"expectPayload": func(json gjson.Result) {
				Expect(json.Map()).To(HaveKey("names"))

				expectedName := gjson.Parse(fmt.Sprintf("%q", createdFeatureFlagName))
				Expect(json.Get("names").Array()).ToNot(ContainElement(expectedName))
			},
		}, url.Values{})
	})

	It("when does not exist, can't be deleted", func() {
		TestAPI.Delete(api.Params{
			"name":         "not-exists",
			"expectStatus": api.ExpectExactStatus(404),
		}, nil)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			createdFeatureFlagName := createRandomFeatureFlag(TestAPI)
			TestAPI.Delete(api.Params{
				"name": createdFeatureFlagName,
			}, nil)

			TestAPI.Delete(api.Params{
				"name":         createdFeatureFlagName,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})
	})
})

func createRandomFeatureFlag(featureFlagAPI api.TestAPI) (featureFlagName string) {
	resp := featureFlagAPI.Create(api.Params{}, url.Values{}, fixtures.RandomFeatureFlagCreatePayload())
	return resp.Get("feature_flag.name").String()
}
