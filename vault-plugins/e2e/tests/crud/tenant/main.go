package tenant

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tenant"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
)

var _ = Describe("Tenant", func() {
	rootClient := lib.GetVaultClient(lib.RootToken)
	tenantsAPI := lib.NewTenantAPI(rootClient)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				var payload struct {
					Identifier interface{} `json:"identifier,omitempty"`
				}
				payload.Identifier = identifier

				params := tools.Params{"expectStatus": tools.ExpectStatus(statusCodeCondition)}

				tenantsAPI.Create(params, nil, payload)
			},
			Entry("number allowed", 100, "%d == 201"),
			Entry("absent identifier forbidden", nil, "%d >= 400"),
			Entry("empty string forbidden", "", "%d >= 400"),
			Entry("array forbidden", []string{"a"}, "%d >= 400"),
			Entry("object forbidden", map[string]int{"a": 1}, "%d >= 400"),
		)
	})

	It("can be created", func() {
		payload := tenant.GetPayload()

		params := tools.Params{
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)

				Expect(data.Map()).To(HaveKey("uuid"))
				Expect(data.Map()).To(HaveKey("identifier"))
				Expect(data.Map()).To(HaveKey("resource_version"))

				Expect(data.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(data.Get("resource_version").String()).ToNot(HaveLen(10))

			},
		}
		tenantsAPI.Create(params, nil, payload)
	})

	It("can be read", func() {
		payload := tenant.GetPayload()

		var createdData gjson.Result
		tenantsAPI.Create(tools.Params{
			"expectPayload": func(b []byte) {
				createdData = tools.UnmarshalVaultResponse(b)
			},
		}, nil, payload)

		tenantsAPI.Read(tools.Params{
			"tenant": createdData.Get("uuid").String(),
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				Expect(createdData).To(Equal(data))
			},
		}, nil)
	})

	It("can be updated", func() {
		createPayload := tenant.GetPayload()

		var createdData gjson.Result
		tenantsAPI.Create(tools.Params{
			"expectPayload": func(b []byte) {
				createdData = tools.UnmarshalVaultResponse(b)
			},
		}, nil, createPayload)

		updatePayload := tenant.GetPayload()
		updatedTenant := model.Tenant{
			UUID:       createdData.Get("uuid").String(),
			Version:    createdData.Get("resource_version").String(),
			Identifier: updatePayload.Identifier.(string),
		}

		var updateData gjson.Result
		tenantsAPI.Update(tools.Params{
			"tenant": createdData.Get("uuid").String(),
			"expectPayload": func(b []byte) {
				updateData = tools.UnmarshalVaultResponse(b)
			},
		}, nil, updatedTenant)

		tenantsAPI.Read(tools.Params{
			"tenant": createdData.Get("uuid").String(),
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				Expect(updateData).To(Equal(data))
			},
		}, nil)
	})

	It("can be deleted", func() {
		createPayload := tenant.GetPayload()

		var createdData gjson.Result
		tenantsAPI.Create(tools.Params{
			"expectPayload": func(b []byte) {
				createdData = tools.UnmarshalVaultResponse(b)
			},
		}, nil, createPayload)

		tenantsAPI.Delete(tools.Params{
			"tenant": createdData.Get("uuid").String(),
		}, nil)

		tenantsAPI.Read(tools.Params{
			"tenant":       createdData.Get("uuid").String(),
			"expectStatus": tools.ExpectExactStatus(404),
		}, nil)
	})

	It("can be listed", func() {
		createPayload := tenant.GetPayload()
		tenantsAPI.Create(tools.Params{}, nil, createPayload)
		tenantsAPI.List(tools.Params{}, nil)
	})
})
