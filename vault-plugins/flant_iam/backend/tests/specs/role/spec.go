package role

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var TestAPI api.TestAPI

var _ = Describe("Role", func() {
	Describe("payload", func() {
		DescribeTable("name",
			func(name interface{}, statusCodeCondition string) {
				createPayload := fixtures.RandomRoleCreatePayload()
				createPayload["name"] = name

				params := api.Params{"expectStatus": api.ExpectStatus(statusCodeCondition)}

				TestAPI.Create(params, nil, createPayload)
			},
			Entry("number is allowed", 100, "%d == 201"),
			Entry("absent identifier forbidden", nil, "%d >= 400"),
			Entry("empty string forbidden", "", "%d >= 400"),
			Entry("array forbidden", []string{"a"}, "%d >= 400"),
			Entry("object forbidden", map[string]int{"a": 1}, "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomRoleCreatePayload()

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				// d := tools.UnmarshalVaultResponse(b)
				data := json.Get("role")

				Expect(data.Map()).To(HaveKey("name"))
				Expect(data.Map()).To(HaveKey("scope"))
				Expect(data.Map()).To(HaveKey("description"))
				Expect(data.Map()).To(HaveKey("options_schema"))
				Expect(data.Map()).To(HaveKey("require_one_of_feature_flags"))
				Expect(data.Map()).To(HaveKey("included_roles"))
				Expect(data.Map()).To(HaveKey("archiving_timestamp"))
				Expect(data.Map()).To(HaveKey("archiving_hash"))
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		role := specs.CreateRandomRole(TestAPI)
		createdData := specs.ConvertToGJSON(role)

		TestAPI.Read(api.Params{
			"name": role.Name,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("role"))
			},
		}, nil)
	})

	It("can be updated", func() {
		role := specs.CreateRandomRole(TestAPI)

		updatedRole := model.Role{
			Name:                     role.Name,
			Scope:                    role.Scope,
			Description:              uuid.New(),
			OptionsSchema:            role.OptionsSchema,
			RequireOneOfFeatureFlags: role.RequireOneOfFeatureFlags,
			IncludedRoles:            role.IncludedRoles,
		}

		updatePayload := fixtures.RoleCreatePayload(updatedRole)

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"name": role.Name,
			"expectPayload": func(json gjson.Result) {
				updateData = json.Get("role")
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"name": role.Name,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json.Get("role"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		role := specs.CreateRandomRole(TestAPI)

		TestAPI.Delete(api.Params{
			"name":         role.Name,
			"expectStatus": api.ExpectExactStatus(http.StatusNoContent),
		}, nil)

		TestAPI.Read(api.Params{
			"name":         role.Name,
			"expectStatus": api.ExpectExactStatus(http.StatusOK),
			"expectPayload": func(json gjson.Result) {
				Expect(json.Get("role.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
			},
		}, nil)
	})
})
