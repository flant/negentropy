package tenant

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var TestAPI api.TestAPI

var _ = Describe("Tenant", func() {
	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				payload := fixtures.RandomTenantCreatePayload()
				payload["identifier"] = identifier

				params := api.Params{"expectStatus": api.ExpectStatus(statusCodeCondition)}

				TestAPI.Create(params, nil, payload)
			},
			Entry("number allowed", 100, "%d == 201"),
			Entry("absent identifier forbidden", nil, "%d >= 400"),
			Entry("empty string forbidden", "", "%d >= 400"),
			Entry("array forbidden", []string{"a"}, "%d >= 400"),
			Entry("object forbidden", map[string]int{"a": 1}, "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomTenantCreatePayload()

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				tenantData := json.Get("tenant")
				Expect(tenantData.Map()).To(HaveKey("uuid"))
				Expect(tenantData.Map()).To(HaveKey("identifier"))
				Expect(tenantData.Map()).To(HaveKey("resource_version"))
				Expect(tenantData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(tenantData.Get("resource_version").String()).ToNot(HaveLen(10))
				Expect(tenantData.Map()).To(HaveKey("origin"))
				Expect(tenantData.Get("origin").String()).To(Equal("iam"))
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		createPayload := fixtures.RandomTenantCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Read(api.Params{
			"tenant": createdData.Get("tenant.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json, "full_restore")
			},
		}, nil)
	})

	It("can be updated", func() {
		createPayload := fixtures.RandomTenantCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		updatePayload := fixtures.RandomTenantCreatePayload()
		updatePayload["resource_version"] = createdData.Get("tenant.resource_version").String()
		updatePayload["identifier"] = createdData.Get("tenant.uuid").String()

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"tenant": createdData.Get("tenant.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant": createdData.Get("tenant.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json, "full_restore")
				tenantData := json.Get("tenant")
				Expect(tenantData.Map()).To(HaveKey("origin"))
				Expect(tenantData.Get("origin").String()).To(Equal("iam"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		createPayload := fixtures.RandomTenantCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Delete(api.Params{
			"tenant": createdData.Get("tenant.uuid").String(),
		}, nil)

		deletedTenantData := TestAPI.Read(api.Params{
			"tenant":       createdData.Get("tenant.uuid").String(),
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedTenantData.Get("tenant.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		createPayload := fixtures.RandomTenantCreatePayload()
		TestAPI.Create(api.Params{}, url.Values{}, createPayload)
		TestAPI.List(api.Params{}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomTenantCreatePayload()
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				tenantData := json.Get("tenant")
				Expect(tenantData.Map()).To(HaveKey("uuid"))
				Expect(tenantData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			tenant := specs.CreateRandomTenant(TestAPI)
			TestAPI.Delete(api.Params{
				"tenant": tenant.UUID,
			}, nil)

			TestAPI.Delete(api.Params{
				"tenant":       tenant.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			tenant := specs.CreateRandomTenant(TestAPI)
			TestAPI.Delete(api.Params{
				"tenant": tenant.UUID,
			}, nil)

			updatePayload := fixtures.RandomTenantCreatePayload()
			updatePayload["resource_version"] = tenant.Version
			updatePayload["identifier"] = tenant.Identifier
			TestAPI.Update(api.Params{
				"tenant":       tenant.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})
