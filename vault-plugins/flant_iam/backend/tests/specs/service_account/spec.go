package serviceaccount

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI            api.TestAPI
	TenantAPI          api.TestAPI
	IdentitySharingAPI api.TestAPI
	GroupAPI           api.TestAPI
)

var _ = Describe("ServiceAccount", func() {
	var (
		tenant                          model.Tenant
		serviceAccount                  model.ServiceAccount
		otherSaOfChildGroupOtherTenant  model.ServiceAccount
		otherSaOfParentGroupOtherTenant model.ServiceAccount
	)
	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
		serviceAccount = specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
		otherTenant := specs.CreateRandomTenant(TenantAPI)
		otherSaOfChildGroupOtherTenant = specs.CreateRandomServiceAccount(TestAPI, otherTenant.UUID)
		otherChildGroup := specs.CreateRandomGroupWithMembers(GroupAPI, otherTenant.UUID, model.Members{
			ServiceAccounts: []string{otherSaOfChildGroupOtherTenant.UUID},
		})
		otherSaOfParentGroupOtherTenant = specs.CreateRandomServiceAccount(TestAPI, otherTenant.UUID)
		otherParentGroup := specs.CreateRandomGroupWithMembers(GroupAPI, otherTenant.UUID, model.Members{
			ServiceAccounts: []string{otherSaOfParentGroupOtherTenant.UUID},
			Groups:          []string{otherChildGroup.UUID},
		})
		specs.ShareGroupToTenant(IdentitySharingAPI, otherParentGroup, tenant.UUID)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomServiceAccountAtTenantWithIdentifier(tenant.UUID, identifier, statusCodeCondition)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomServiceAccountCreatePayload()

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				saData := json.Get("service_account")
				Expect(saData.Map()).To(HaveKey("uuid"))
				Expect(saData.Map()).To(HaveKey("identifier"))
				Expect(saData.Map()).To(HaveKey("full_identifier"))
				Expect(saData.Map()).To(HaveKey("allowed_cidrs"))
				Expect(saData.Map()).To(HaveKey("origin"))
				Expect(saData.Get("origin").String()).To(Equal("iam"))
				Expect(saData.Map()).To(HaveKey("token_ttl"))
				Expect(saData.Map()).To(HaveKey("token_max_ttl"))
				Expect(saData.Get("uuid").String()).To(HaveLen(36))
				Expect(saData.Get("resource_version").String()).To(HaveLen(36))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	Context("tenant uniqueness of serviceAccount identifier", func() {
		identifier := uuid.New()
		It("Can be created serviceAccount with some identifier", func() {
			tryCreateRandomServiceAccountAtTenantWithIdentifier(tenant.UUID, identifier, "%d == 201")
		})
		It("Can not be the same identifier at the same tenant", func() {
			tryCreateRandomServiceAccountAtTenantWithIdentifier(tenant.UUID, identifier, "%d >= 400")
		})
		It("Can be same identifier at other tenant", func() {
			tenant2 := specs.CreateRandomTenant(TenantAPI)
			tryCreateRandomServiceAccountAtTenantWithIdentifier(tenant2.UUID, identifier, "%d == 201")
		})
	})

	It("can be read", func() {
		createdData := specs.ConvertToGJSON(serviceAccount)

		TestAPI.Read(api.Params{
			"tenant":          serviceAccount.TenantUUID,
			"service_account": serviceAccount.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("service_account"), "extensions")
			},
		}, nil)
	})

	It("can be updated", func() {
		sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
		updatePayload := fixtures.RandomServiceAccountCreatePayload()
		updatePayload["resource_version"] = sa.Version

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json, "full_restore")
				saData := json.Get("service_account")
				Expect(saData.Map()).To(HaveKey("origin"))
				Expect(saData.Get("origin").String()).To(Equal("iam"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)

		deleteServiceAccount(sa)

		deletedData := TestAPI.Read(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
			"expectStatus":    api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("service_account.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	Context("can be listed", func() {
		It("result contains only own service_accounts if not passed shared=true", func() {
			TestAPI.List(api.Params{
				"tenant": serviceAccount.TenantUUID,
				"expectPayload": func(json gjson.Result) {
					sasArray := json.Get("service_accounts").Array()
					specs.CheckArrayContainsElementByUUIDExceptKeys(sasArray,
						specs.ConvertToGJSON(serviceAccount), "extensions")
					specs.CheckObjectArrayForUUID(sasArray, otherSaOfChildGroupOtherTenant.UUID, false)
					specs.CheckObjectArrayForUUID(sasArray, otherSaOfParentGroupOtherTenant.UUID, false)
				},
			}, url.Values{})
		})

		It("result contains shared service_accounts if  passed shared=true", func() {
			TestAPI.List(api.Params{
				"tenant": serviceAccount.TenantUUID,
				"expectPayload": func(json gjson.Result) {
					sasArray := json.Get("service_accounts").Array()
					specs.CheckArrayContainsElementByUUIDExceptKeys(sasArray,
						specs.ConvertToGJSON(serviceAccount), "extensions")
					specs.CheckObjectArrayForUUID(sasArray, otherSaOfChildGroupOtherTenant.UUID, true)
					specs.CheckObjectArrayForUUID(sasArray, otherSaOfParentGroupOtherTenant.UUID, true)
				},
			}, url.Values{"show_shared": []string{"true"}})
		})
	})
	// It("can not be created with privileged", func() {

	Context("after deletion", func() {
		It("can't be deleted", func() {
			sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
			deleteServiceAccount(sa)

			TestAPI.Delete(api.Params{
				"tenant":          sa.TenantUUID,
				"service_account": sa.UUID,
				"expectStatus":    api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
			deleteServiceAccount(sa)

			updatePayload := fixtures.RandomServiceAccountCreatePayload()
			updatePayload["tenant_uuid"] = sa.TenantUUID
			updatePayload["resource_version"] = sa.Version
			TestAPI.Update(api.Params{
				"tenant":          sa.TenantUUID,
				"service_account": sa.UUID,
				"expectStatus":    api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})

	Context("restoring deleted service_account", func() {
		It("can't be restored after deleting", func() {
			serviceAccount := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
			deleteServiceAccount(serviceAccount)

			TestAPI.Restore(api.Params{
				"tenant":          serviceAccount.TenantUUID,
				"service_account": serviceAccount.UUID,
				"expectStatus":    api.ExpectStatus("%d > 400"),
			}, nil)
		})
	})
})

func tryCreateRandomServiceAccountAtTenantWithIdentifier(tenantUUID string,
	serviceAccountIdentifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomServiceAccountCreatePayload()
	payload["identifier"] = serviceAccountIdentifier

	params := api.Params{
		"tenant":       tenantUUID,
		"expectStatus": api.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}

func deleteServiceAccount(service_account model.ServiceAccount) {
	TestAPI.Delete(api.Params{
		"expectStatus":    api.ExpectExactStatus(http.StatusNoContent),
		"tenant":          service_account.TenantUUID,
		"service_account": service_account.UUID,
	}, nil)
}
