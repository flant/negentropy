package identitysharing

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TenantAPI api.TestAPI
	UserAPI   api.TestAPI
	GroupAPI  api.TestAPI
	TestAPI   api.TestAPI
)

var _ = Describe("Identity sharing", func() {
	var (
		sourceTenantID, targetTenantID string
		user                           model.User
		group                          model.Group
	)

	BeforeEach(func() {
		t1 := specs.CreateRandomTenant(TenantAPI)
		sourceTenantID = t1.UUID
		t2 := specs.CreateRandomTenant(TenantAPI)
		targetTenantID = t2.UUID
		user = specs.CreateRandomUser(UserAPI, sourceTenantID)
		group = specs.CreateRandomGroupWithUser(GroupAPI, sourceTenantID, user.UUID)
	})

	var createdData gjson.Result

	It("can be created", func() {
		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				is := json.Get("identity_sharing")

				Expect(is.Map()).To(HaveKey("uuid"))
				Expect(is.Map()).To(HaveKey("source_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("destination_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("groups"))
				Expect(is.Map()).To(HaveKey("origin"))
				Expect(is.Get("groups").Array()).To(HaveLen(1))
			},
			"tenant": sourceTenantID,
		}
		data := map[string]interface{}{
			"destination_tenant_uuid": targetTenantID,
			"groups":                  []string{group.UUID},
		}
		createdData = TestAPI.Create(params, url.Values{}, data)
	})

	It("can be read", func() {
		TestAPI.Read(api.Params{
			"uuid":   createdData.Get("identity_sharing.uuid").String(),
			"tenant": sourceTenantID,
			"expectPayload": func(json gjson.Result) {
				Expect(createdData).To(Equal(json))
			},
		}, nil)
	})

	It("can be listed", func() {
		createdISUUID := createIdentitySharing(TestAPI, targetTenantID, group)
		list := TestAPI.List(api.Params{
			"tenant": sourceTenantID,
		}, url.Values{})
		Expect(list.Get("identity_sharings").Array()).To(HaveLen(1))
		Expect(list.Get("identity_sharings").Array()[0].Get("uuid").String()).To(BeEquivalentTo(createdISUUID))
	})

	It("can be deleted", func() {
		createdISUUID := createIdentitySharing(TestAPI, targetTenantID, group)
		TestAPI.Delete(api.Params{
			"uuid":   createdISUUID,
			"tenant": sourceTenantID,
		}, nil)

		deletedISData := TestAPI.Read(api.Params{
			"uuid":         createdISUUID,
			"tenant":       sourceTenantID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedISData.Get("identity_sharing.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be created with privileged", func() {
		t1 := specs.CreateRandomTenant(TenantAPI)
		sourceTenantID = t1.UUID
		t2 := specs.CreateRandomTenant(TenantAPI)
		targetTenantID = t2.UUID
		user = specs.CreateRandomUser(UserAPI, sourceTenantID)
		group = specs.CreateRandomGroupWithUser(GroupAPI, sourceTenantID, user.UUID)

		originalUUID := uuid.New()

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				is := json.Get("identity_sharing")

				Expect(is.Map()).To(HaveKey("uuid"))
				Expect(is.Map()["uuid"].String()).To(Equal(originalUUID))
				Expect(is.Map()).To(HaveKey("source_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("destination_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("groups"))
				Expect(is.Get("groups").Array()).To(HaveLen(1))
			},
			"tenant": sourceTenantID,
		}
		data := map[string]interface{}{
			"destination_tenant_uuid": targetTenantID,
			"groups":                  []string{group.UUID},
			"uuid":                    originalUUID,
		}
		createdData = TestAPI.CreatePrivileged(params, url.Values{}, data)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			isUUID := createIdentitySharing(TestAPI, targetTenantID, group)
			TestAPI.Delete(api.Params{
				"uuid":   isUUID,
				"tenant": sourceTenantID,
			}, nil)

			TestAPI.Delete(api.Params{
				"uuid":         isUUID,
				"tenant":       sourceTenantID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		// NO update path for identity_sharing
	})
})

func createIdentitySharing(identitySharingAPI api.TestAPI, targetTenantID string, group model.Group) string {
	data := map[string]interface{}{
		"destination_tenant_uuid": targetTenantID,
		"groups":                  []string{group.UUID},
	}
	createdData := identitySharingAPI.Create(api.Params{
		"tenant": group.TenantUUID,
	}, url.Values{}, data)
	return createdData.Get("identity_sharing.uuid").String()
}
