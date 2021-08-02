package tenant

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/featureflag"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
)

var _ = Describe("Feature Flag", func() {
	rootClient := lib.NewConfiguredIamVaultClient()
	flagsAPI := lib.NewFeatureFlagAPI(rootClient)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(name interface{}, statusCodeCondition string) {
				var payload featureflag.Payload
				payload.Name = name

				params := tools.Params{"expectStatus": tools.ExpectStatus(statusCodeCondition)}

				flagsAPI.Create(params, nil, payload)
			},
			Entry("number allowed", rand.Intn(32), "%d == 201"), // the matter of fact ¯\_(ツ)_/¯
			Entry("absent identifier forbidden", nil, "%d >= 400"),
			Entry("empty string forbidden", "", "%d >= 400"),
			Entry("array forbidden", []string{"a"}, "%d >= 400"),
			Entry("object forbidden", map[string]int{"a": 1}, "%d >= 400"),
		)
	})

	It("can be created", func() {
		payload := featureflag.GetPayload()

		params := tools.Params{
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				Expect(data.Get("feature_flag").Map()).To(HaveKey("name"))
			},
		}
		flagsAPI.Create(params, url.Values{}, payload)
	})

	It("can be listed", func() {
		createPayload := featureflag.GetPayload()
		flagsAPI.Create(tools.Params{}, url.Values{}, createPayload)
		flagsAPI.List(tools.Params{}, url.Values{})
	})

	It("has identifying fields in list", func() {
		createPayload := featureflag.GetPayload()
		flagsAPI.Create(tools.Params{}, url.Values{}, createPayload)

		params := tools.Params{
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				Expect(data.Map()).To(HaveKey("names"))
				expectedFF := gjson.Parse(fmt.Sprintf("{\"archiving_hash\":0,\"archiving_timestamp\":0,\"name\":\"%s\"}", createPayload.Name))
				Expect(data.Get("names").Array()).To(ContainElement(expectedFF))
			},
		}
		flagsAPI.List(params, url.Values{})
	})

	It("can be deleted", func() {
		createPayload := featureflag.GetPayload()

		var createdData gjson.Result
		flagsAPI.Create(tools.Params{
			"expectPayload": func(b []byte) {
				createdData = tools.UnmarshalVaultResponse(b)
			},
		}, nil, createPayload)

		flagsAPI.Delete(tools.Params{
			"name": createdData.Get("feature_flag.name").String(),
		}, nil)

		flagsAPI.List(tools.Params{
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				Expect(data.Map()).To(HaveKey("names"))

				expectedName := gjson.Parse(fmt.Sprintf("%q", createdData.Get("name").String()))
				Expect(data.Get("names").Array()).ToNot(ContainElement(expectedName))
			},
		}, url.Values{})
	})

	It("when does not exist", func() {
		flagsAPI.Delete(tools.Params{
			"name":         "not-exists",
			"expectStatus": tools.ExpectExactStatus(404),
		}, nil)
	})

	Describe("no access", func() {
		runWithClient := func(client *http.Client, expectedStatus func(response *http.Response)) {
			flagsAPI := lib.NewFeatureFlagAPI(client)

			params := tools.Params{"expectStatus": expectedStatus}

			It("Cannot create", func() {
				createPayload := featureflag.GetPayload()
				flagsAPI.Create(params, url.Values{}, createPayload)
			})

			It("Cannot list", func() {
				flagsAPI.List(tools.Params{"expectStatus": expectedStatus}, url.Values{})
			})

			It("Cannot read", func() {
				createPayload := featureflag.GetPayload()
				flagsAPI.Create(params, url.Values{}, createPayload)
				params["name"] = createPayload.Name.(string)
				flagsAPI.Read(params, nil)
			})

			It("Cannot update", func() {
				createPayload := featureflag.GetPayload()
				flagsAPI.Create(params, url.Values{}, createPayload)
				params["name"] = createPayload.Name.(string)
				flagsAPI.Update(params, nil, nil)
			})

			It("Cannot delete", func() {
				createPayload := featureflag.GetPayload()
				flagsAPI.Create(params, url.Values{}, createPayload)
				params["name"] = createPayload.Name.(string)
				flagsAPI.Delete(params, nil)
			})
		}

		Describe("when unauthenticated", func() {
			runWithClient(lib.NewIamVaultClient(""), tools.ExpectExactStatus(400))
		})

		Describe("when unauthorized", func() {
			runWithClient(lib.NewIamVaultClient("xxx"), tools.ExpectExactStatus(403))
		})
	})
})
