package rolebindingapproval

import (
	"encoding/json"
	"fmt"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TenantAPI         api.TestAPI
	UserAPI           api.TestAPI
	ServiceAccountAPI api.TestAPI
	GroupAPI          api.TestAPI
	ProjectAPI        api.TestAPI
	RoleAPI           api.TestAPI
	RoleBindingAPI    api.TestAPI
	TestAPI           api.TestAPI
)

var _ = Describe("Role binding approval", func() {
	var (
		tenant model.Tenant
		user   model.User
		sa     model.ServiceAccount
		group  model.Group
	)

	var roleBindingID string

	var createdRBA gjson.Result
	var updatedRBA gjson.Result

	rbaaCreator := func() {
		approvers := []map[string]interface{}{
			{"type": "user", "uuid": user.UUID},
			{"type": "service_account", "uuid": sa.UUID},
			{"type": "group", "uuid": group.UUID},
		}

		data := map[string]interface{}{
			"required_votes": 3,
			"approvers":      approvers,
		}

		params := api.Params{
			"expectStatus": api.ExpectExactStatus(201),
			"expectPayload": func(js gjson.Result) {
				ap := js.Get("approval")
				Expect(ap.Get("required_votes").Int()).To(BeEquivalentTo(3))
				Expect(ap.Get("approvers").Array()).To(HaveLen(3))
				checkApprovers(approvers, ap)
			},
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
			"uuid":         uuid.New(),
		}

		createdData := TestAPI.Create(params, url.Values{}, data)
		createdRBA = createdData.Get("approval")
	}

	BeforeSuite(func() {
		specs.CreateRoles(RoleAPI, fixtures.Roles()...)
		tenant = specs.CreateRandomTenant(TenantAPI)
		user = specs.CreateRandomUser(UserAPI, tenant.UUID)
		sa = specs.CreateRandomServiceAccount(ServiceAccountAPI, tenant.UUID)
		group = specs.CreateRandomGroupWithUser(GroupAPI, tenant.UUID, user.UUID)
		res := RoleBindingAPI.Create(api.Params{"tenant": tenant.UUID}, url.Values{}, fixtures.RandomRoleBindingCreatePayloadWithUser(user.UUID))
		roleBindingID = res.Get("role_binding.uuid").String()
		rbaaCreator()
	})

	It("can be created", func() {
		rbaaCreator()
	})

	It("can be read", func() {
		// Created before
		TestAPI.Read(api.Params{
			"uuid":         createdRBA.Get("uuid").String(),
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
			"expectPayload": func(json gjson.Result) {
				Expect(createdRBA).To(Equal(json.Get("approval")))
			},
		}, nil)
	})

	It("can be updated", func() {
		// Created before
		approvers := []map[string]interface{}{
			{"type": "service_account", "uuid": sa.UUID},
			{"type": "group", "uuid": group.UUID},
		}
		updatePayload := map[string]interface{}{
			"required_votes":   3,
			"approvers":        approvers,
			"resource_version": createdRBA.Get("resource_version").String(),
		}

		updatedData := TestAPI.Update(api.Params{
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
			"uuid":         createdRBA.Get("uuid").String(),
			"expectPayload": func(js gjson.Result) {
				ap := js.Get("approval")
				Expect(ap.Get("required_votes").Int()).To(BeEquivalentTo(3))
				Expect(ap.Get("approvers").Array()).To(HaveLen(2))
				checkApprovers(approvers, ap)
			},
		}, nil, updatePayload)
		updatedRBA = updatedData.Get("approval")
	})

	It("can be deleted", func() {
		// Created before
		TestAPI.Delete(api.Params{
			"uuid":         createdRBA.Get("uuid").String(),
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
		}, nil)

		deletedRBData := TestAPI.Read(api.Params{
			"uuid":         createdRBA.Get("uuid").String(),
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedRBData.Get("approval.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			// created and deleted before
			TestAPI.Delete(api.Params{
				"uuid":         createdRBA.Get("uuid").String(),
				"tenant":       tenant.UUID,
				"role_binding": roleBindingID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			// created and deleted before
			updatePayload := map[string]interface{}{
				"required_votes": 3,
				"approvers": []map[string]interface{}{
					{"type": "group", "uuid": group.UUID},
				},
				"resource_version": updatedRBA.Get("resource_version").String(),
			}

			TestAPI.Update(api.Params{
				"tenant":       tenant.UUID,
				"role_binding": roleBindingID,
				"uuid":         createdRBA.Get("uuid").String(),
				"expectStatus": api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func checkApprovers(approvers []map[string]interface{}, approval gjson.Result) {
	approversBytes, err := json.Marshal(approvers)
	Expect(err).ToNot(HaveOccurred())
	approversStr := string(approversBytes)
	approversFromGjson := approval.Get("approvers").String()

	Expect(specs.AreEqualJSON(approversStr, approversFromGjson)).To(BeTrue(), fmt.Sprintf("%s should be equal %s", approversStr, approversFromGjson))
}
