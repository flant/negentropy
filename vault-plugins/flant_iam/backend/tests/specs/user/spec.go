package user

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

var _ = Describe("User", func() {
	var (
		tenant                            model.Tenant
		otherUserOfChildGroupOtherTenant  model.User
		otherUserOfParentGroupOtherTenant model.User
	)

	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
		otherTenant := specs.CreateRandomTenant(TenantAPI)
		otherUserOfChildGroupOtherTenant = specs.CreateRandomUser(TestAPI, otherTenant.UUID)
		otherChildGroup := specs.CreateRandomGroupWithMembers(GroupAPI, otherTenant.UUID, model.Members{
			Users: []string{otherUserOfChildGroupOtherTenant.UUID},
		})
		otherUserOfParentGroupOtherTenant = specs.CreateRandomUser(TestAPI, otherTenant.UUID)
		otherParentGroup := specs.CreateRandomGroupWithMembers(GroupAPI, otherTenant.UUID, model.Members{
			Users:  []string{otherUserOfParentGroupOtherTenant.UUID},
			Groups: []string{otherChildGroup.UUID},
		})
		specs.ShareGroupToTenant(IdentitySharingAPI, otherParentGroup, tenant.UUID)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomUserAtTenantWithIdentifier(tenant.UUID, identifier, statusCodeCondition)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomUserCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				userData := json.Get("user")
				Expect(userData.Map()).To(HaveKey("uuid"))
				Expect(userData.Map()).To(HaveKey("identifier"))
				Expect(userData.Map()).To(HaveKey("full_identifier"))
				Expect(userData.Map()).To(HaveKey("email"))
				Expect(userData.Map()).To(HaveKey("origin"))
				Expect(userData.Get("origin").String()).To(Equal("iam"))
				Expect(userData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(userData.Get("resource_version").String()).ToNot(HaveLen(10))
				Expect(userData.Map()).To(HaveKey("language"))
				Expect(userData.Get("language").String()).To(Equal(createPayload["language"]))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	Context("tenant uniqueness of user identifier", func() {
		identifier := uuid.New()
		It("Can be created user with some identifier", func() {
			tryCreateRandomUserAtTenantWithIdentifier(tenant.UUID, identifier, "%d == 201")
		})
		It("Can not be the same identifier at the same tenant", func() {
			tryCreateRandomUserAtTenantWithIdentifier(tenant.UUID, identifier, "%d >= 400")
		})
		It("Can be same identifier at other tenant", func() {
			tenant2 := specs.CreateRandomTenant(TenantAPI)
			tryCreateRandomUserAtTenantWithIdentifier(tenant2.UUID, identifier, "%d == 201")
		})
	})

	It("can be read", func() {
		user := specs.CreateRandomUser(TestAPI, tenant.UUID)
		createdData := specs.ConvertToGJSON(user)

		TestAPI.Read(api.Params{
			"tenant": user.TenantUUID,
			"user":   user.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("user"), "extensions")
			},
		}, nil)
	})

	It("can be updated", func() {
		user := specs.CreateRandomUser(TestAPI, tenant.UUID)
		updatePayload := fixtures.RandomUserCreatePayload()
		updatePayload["tenant_uuid"] = user.TenantUUID
		updatePayload["resource_version"] = user.Version

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"tenant": user.TenantUUID,
			"user":   user.UUID,
			"expectPayload": func(json gjson.Result) {
				updateData = json
				userData := updateData.Get("user")
				Expect(userData.Map()).To(HaveKey("origin"))
				Expect(userData.Get("origin").String()).To(Equal("iam"))
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant": user.TenantUUID,
			"user":   user.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json, "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		user := specs.CreateRandomUser(TestAPI, tenant.UUID)

		deleteUser(user)

		deletedData := TestAPI.Read(api.Params{
			"tenant":       user.TenantUUID,
			"user":         user.UUID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("user.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	Context("can be listed", func() {
		It("result contains only own users if not passed shared=true", func() {
			user := specs.CreateRandomUser(TestAPI, tenant.UUID)

			TestAPI.List(api.Params{
				"tenant": user.TenantUUID,
				"expectPayload": func(json gjson.Result) {
					usersArray := json.Get("users").Array()
					specs.CheckArrayContainsElementByUUIDExceptKeys(usersArray,
						specs.ConvertToGJSON(user), "extensions")
					specs.CheckObjectArrayForUUID(usersArray, otherUserOfChildGroupOtherTenant.UUID, false)
					specs.CheckObjectArrayForUUID(usersArray, otherUserOfParentGroupOtherTenant.UUID, false)
				},
			}, url.Values{})
		})

		It("result contains shared users if passed show_shared=true", func() {
			user := specs.CreateRandomUser(TestAPI, tenant.UUID)

			TestAPI.List(api.Params{
				"tenant": user.TenantUUID,
				"expectPayload": func(json gjson.Result) {
					usersArray := json.Get("users").Array()
					specs.CheckArrayContainsElementByUUIDExceptKeys(usersArray,
						specs.ConvertToGJSON(user), "extensions")
					specs.CheckObjectArrayForUUID(usersArray, otherUserOfChildGroupOtherTenant.UUID, true)
					specs.CheckObjectArrayForUUID(usersArray, otherUserOfParentGroupOtherTenant.UUID, true)
				},
			}, url.Values{"show_shared": []string{"true"}})
		})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomUserCreatePayload()
		originalUUID := createPayload["uuid"]
		createPayload["tenant_uuid"] = tenant.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				userData := json.Get("user")
				Expect(userData.Map()).To(HaveKey("uuid"))
				Expect(userData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			user := specs.CreateRandomUser(TestAPI, tenant.UUID)
			deleteUser(user)

			TestAPI.Delete(api.Params{
				"tenant":       user.TenantUUID,
				"user":         user.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			user := specs.CreateRandomUser(TestAPI, tenant.UUID)
			deleteUser(user)

			updatePayload := fixtures.RandomUserCreatePayload()
			updatePayload["tenant_uuid"] = user.TenantUUID
			updatePayload["resource_version"] = user.Version
			TestAPI.Update(api.Params{
				"tenant":       user.TenantUUID,
				"user":         user.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})

	Context("restoring deleted user", func() {
		It("can be restored after deleting", func() {
			user := specs.CreateRandomUser(TestAPI, tenant.UUID)
			deleteUser(user)

			TestAPI.Restore(api.Params{
				"tenant":       user.TenantUUID,
				"user":         user.UUID,
				"expectStatus": api.ExpectExactStatus(200),
				"expectPayload": func(json gjson.Result) {
					userData := json.Get("user")
					Expect(userData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically("==", int64(0))))
				},
			}, nil)
		})

		It("cant be restored after deleting tenant", func() {
			otherTenant := specs.CreateRandomTenant(TenantAPI)
			user := specs.CreateRandomUser(TestAPI, otherTenant.UUID)
			deleteUser(user)
			TenantAPI.Delete(api.Params{
				"expectStatus": api.ExpectExactStatus(http.StatusNoContent),
				"tenant":       otherTenant.UUID,
			}, nil)

			TestAPI.Restore(api.Params{
				"tenant":       user.TenantUUID,
				"user":         user.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)

			TestAPI.Read(api.Params{
				"tenant":       user.TenantUUID,
				"user":         user.UUID,
				"expectStatus": api.ExpectExactStatus(200),
				"expectPayload": func(json gjson.Result) {
					userData := json.Get("user")
					Expect(userData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
				},
			}, nil)
		})
	})
})

func tryCreateRandomUserAtTenantWithIdentifier(tenantUUID string,
	userIdentifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomUserCreatePayload()
	payload["identifier"] = userIdentifier

	params := api.Params{
		"tenant":       tenantUUID,
		"expectStatus": api.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}

func deleteUser(user model.User) {
	TestAPI.Delete(api.Params{
		"expectStatus": api.ExpectExactStatus(http.StatusNoContent),
		"tenant":       user.TenantUUID,
		"user":         user.UUID,
	}, nil)
}
