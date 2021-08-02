package specs

import (
	"net/url"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

var RoleAPI api.TestAPI

var _ = Describe("Role", func() {
	Describe("payload", func() {
		DescribeTable("name",
			func(name interface{}, statusCodeCondition string) {
				role := fixtures.Roles()[0]
				createPayload := fixtures.RoleCreatePayload(role)
				createPayload["name"] = name

				params := api.Params{"expectStatus": api.ExpectStatus(statusCodeCondition)}

				RoleAPI.Create(params, nil, createPayload)
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
		RoleAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		createPayload := fixtures.RandomRoleCreatePayload()

		var createdData gjson.Result
		RoleAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		RoleAPI.Read(api.Params{
			"name": createdData.Get("role.name").String(),
			"expectPayload": func(json gjson.Result) {
				Expect(createdData).To(Equal(json))
			},
		}, nil)
	})

	It("can be updated", func() {
		createPayload := fixtures.RandomRoleCreatePayload()

		var createdData gjson.Result
		RoleAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		updatedRole := model.Role{
			Name:                     createdData.Get("role.name").String(),
			Scope:                    model.RoleScope(createdData.Get("role.scope").String()),
			Description:              uuid.New(),
			OptionsSchema:            createdData.Get("role.option_schema").String(),
			RequireOneOfFeatureFlags: []string{},
			IncludedRoles:            []model.IncludedRole{},
			ArchivingTimestamp:       createdData.Get("role.archiving_timestamp").Int(),
			ArchivingHash:            createdData.Get("role.archiving_hash").Int(),
		}

		updatePayload := fixtures.RoleCreatePayload(updatedRole)

		var updateData gjson.Result
		RoleAPI.Update(api.Params{
			"name": createdData.Get("role.name").String(),
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		RoleAPI.Read(api.Params{
			"name": createdData.Get("role.name").String(),
			"expectPayload": func(json gjson.Result) {
				Expect(updateData).To(Equal(json))
			},
		}, nil)
	})

	It("can be deleted", func() {
		createPayload := fixtures.RandomRoleCreatePayload()
		var createdData gjson.Result
		RoleAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		RoleAPI.Delete(api.Params{
			"name": createdData.Get("role.name").String(),
		}, nil)

		RoleAPI.Read(api.Params{
			"name":         createdData.Get("role.name").String(),
			"expectStatus": api.ExpectExactStatus(200),
			"expectPayload": func(json gjson.Result) {
				Expect(json.Get("role.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
			},
		}, nil)
	})
})
