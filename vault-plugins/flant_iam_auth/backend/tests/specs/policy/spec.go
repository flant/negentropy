package policy

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	iam_fixtures "github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/fixtures"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI tests.TestAPI
	RoleAPI tests.TestAPI // TODO REMOVE after put some polciies into migrations
)

var _ = Describe("Policy", func() {
	BeforeSuite(func() {
		specs.CreateRoles(RoleAPI, iam_fixtures.Roles()...) // TODO REMOVE after put some polciies into migrations
	})

	It("can be created", func() {
		createPayload := fixtures.RandomPolicyCreatePayload()

		params := tests.Params{
			"expectPayload": func(json gjson.Result) {
				policyData := json.Get("policy")
				Expect(policyData.Map()).To(HaveKey("name"))
				Expect(policyData.Map()).To(HaveKey("rego"))
				Expect(policyData.Map()).To(HaveKey("roles"))
				Expect(policyData.Map()).To(HaveKey("options_schema"))
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	Context("global uniqueness of policy Name", func() {
		It("Can not be the same Name", func() {
			name := uuid.New()
			tryCreateRandomPolicyWithName(name, "%d == 201")
			tryCreateRandomPolicyWithName(name, "%d >= 400")
		})
	})

	It("can be read", func() {
		createPayload := fixtures.RandomPolicyCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(tests.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Read(tests.Params{
			"policy": createdData.Get("policy.name").String(),
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json)
			},
		}, nil)
	})

	It("can be updated", func() {
		createPayload := fixtures.RandomPolicyCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(tests.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		updatePayload := fixtures.RandomPolicyCreatePayload()

		var updateData gjson.Result
		TestAPI.Update(tests.Params{
			"policy": createdData.Get("policy.name").String(),
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(tests.Params{
			"policy": createdData.Get("policy.name").String(),
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json)
			},
		}, nil)
	})

	It("can be deleted", func() {
		createPayload := fixtures.RandomPolicyCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(tests.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Delete(tests.Params{
			"policy": createdData.Get("policy.name").String(),
		}, nil)

		deletedPolicyData := TestAPI.Read(tests.Params{
			"policy":       createdData.Get("policy.name").String(),
			"expectStatus": tests.ExpectExactStatus(200),
		}, nil)
		Expect(deletedPolicyData.Get("policy.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		createPayload := fixtures.RandomPolicyCreatePayload()
		TestAPI.Create(tests.Params{}, url.Values{}, createPayload)
		TestAPI.List(tests.Params{}, url.Values{})
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			createPayload := fixtures.RandomPolicyCreatePayload()
			var createdData gjson.Result
			TestAPI.Create(tests.Params{
				"expectPayload": func(json gjson.Result) {
					createdData = json
				},
			}, nil, createPayload)
			TestAPI.Delete(tests.Params{
				"policy": createdData.Get("policy.name").String(),
			}, nil)

			TestAPI.Delete(tests.Params{
				"policy":       createdData.Get("policy.name").String(),
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			createPayload := fixtures.RandomPolicyCreatePayload()
			var createdData gjson.Result
			TestAPI.Create(tests.Params{
				"expectPayload": func(json gjson.Result) {
					createdData = json
				},
			}, nil, createPayload)
			TestAPI.Delete(tests.Params{
				"policy": createdData.Get("policy.name").String(),
			}, nil)

			updatePayload := fixtures.RandomPolicyCreatePayload()
			TestAPI.Update(tests.Params{
				"policy":       createdData.Get("policy.name").String(),
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func tryCreateRandomPolicyWithName(name interface{}, statusCodeCondition string) {
	payload := fixtures.RandomPolicyCreatePayload()
	payload["name"] = name

	params := tests.Params{
		"expectStatus": tests.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}
