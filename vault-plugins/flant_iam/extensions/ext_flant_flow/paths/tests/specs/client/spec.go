package client

import (
	"fmt"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI            tests.TestAPI
	RoleAPI            tests.TestAPI
	TeamAPI            tests.TestAPI
	GroupAPI           tests.TestAPI
	ConfigAPI          testapi.ConfigAPI
	UserAPI            tests.TestAPI
	IdentitySharingAPI tests.TestAPI
	RoleBindingAPI     tests.TestAPI
)

var _ = Describe("Client", func() {
	var flantFlowCfg *config.FlantFlowConfig
	var flantUser model.User
	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(RoleAPI, TeamAPI, ConfigAPI)
		flantUser = iam_specs.CreateRandomUser(UserAPI, flantFlowCfg.FlantTenantUUID)
		fmt.Printf("cfg:\n%#v\n", flantFlowCfg)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomClientWithIdentifier(identifier, flantUser.UUID, statusCodeCondition)
			},
			Entry("absent identifier forbidden", nil, "%d >= 400"),
			Entry("empty string forbidden", "", "%d >= 400"),
			Entry("array forbidden", []string{"a"}, "%d >= 400"),
			Entry("object forbidden", map[string]int{"a": 1}, "%d >= 400"),
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomClientCreatePayload()
		createPayload["primary_administrators"] = []string{flantUser.UUID}
		clientUUID := ""

		params := tests.Params{
			"expectPayload": func(json gjson.Result) {
				clientData := json.Get("client")
				Expect(clientData.Map()).To(HaveKey("uuid"))
				clientUUID = clientData.Get("uuid").String()
				Expect(clientData.Map()).To(HaveKey("identifier"))
				Expect(clientData.Map()).To(HaveKey("resource_version"))
				Expect(clientData.Get("uuid").String()).To(HaveLen(36))
				Expect(clientData.Get("resource_version").String()).To(HaveLen(36))
				Expect(clientData.Map()).ToNot(HaveKey("origin"))
				Expect(clientData.Map()).To(HaveKey("language"))
				Expect(clientData.Get("language").String()).To(Equal(createPayload["language"]))
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)

		// Check identity_sharing all flant is created
		checkIdentitySharingAllFlantGroupExists(flantFlowCfg, clientUUID, true)

		// Check primary_admins rolebinding and identity sharing exists
		clientIdentifier := createPayload["identifier"].(string)
		checkPrimaryAdminsStaffExists(flantFlowCfg, clientUUID, clientIdentifier, flantUser.UUID, true)
	})

	Context("global uniqueness of client Identifier", func() {
		It("Can not be the same Identifier", func() {
			identifier := uuid.New()
			tryCreateRandomClientWithIdentifier(identifier, flantUser.UUID, "%d == 201")
			tryCreateRandomClientWithIdentifier(identifier, flantUser.UUID, "%d >= 400")
		})
	})

	It("can be read", func() {
		createPayload := fixtures.RandomClientCreatePayload()
		createPayload["primary_administrators"] = []string{flantUser.UUID}

		var createdData gjson.Result
		TestAPI.Create(tests.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Read(tests.Params{
			"client": createdData.Get("client.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(createdData, json, "full_restore")
				Expect(json.Get("client").Map()).ToNot(HaveKey("origin"))
			},
		}, nil)
	})

	It("can be updated", func() {
		createPayload := fixtures.RandomClientCreatePayload()
		createPayload["primary_administrators"] = []string{flantUser.UUID}

		var createdData gjson.Result
		TestAPI.Create(tests.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		updatePayload := fixtures.RandomClientCreatePayload()
		updatePayload["resource_version"] = createdData.Get("client.resource_version").String()

		TestAPI.Update(tests.Params{
			"client": createdData.Get("client.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				clientData := json.Get("client")
				iam_specs.IsMapSubsetOfSetExceptKeys(updatePayload, clientData, "archiving_timestamp",
					"archiving_hash", "uuid", "resource_version", "origin", "feature_flags")
				Expect(clientData.Map()).ToNot(HaveKey("origin"))
			},
		}, nil, updatePayload)

		TestAPI.Read(tests.Params{
			"client": createdData.Get("client.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				clientData := json.Get("client")
				iam_specs.IsMapSubsetOfSetExceptKeys(updatePayload, clientData, "archiving_timestamp",
					"archiving_hash", "uuid", "resource_version", "origin", "feature_flags")
			},
		}, nil)
	})

	It("can be deleted", func() {
		createdClient := specs.CreateRandomClient(TestAPI, flantUser.UUID)

		TestAPI.Delete(tests.Params{
			"client": createdClient.UUID,
		}, nil)

		deletedClientData := TestAPI.Read(tests.Params{
			"client":       createdClient.UUID,
			"expectStatus": tests.ExpectExactStatus(200),
		}, nil)
		Expect(deletedClientData.Get("client.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))

		// Check identity_sharing is deleted
		checkIdentitySharingAllFlantGroupExists(flantFlowCfg, createdClient.UUID, false)

		// Check sharing & rb is deleted
		checkPrimaryAdminsStaffExists(flantFlowCfg, createdClient.UUID, createdClient.Identifier, flantUser.UUID, false)
	})

	It("can be listed", func() {
		createdClient := specs.CreateRandomClient(TestAPI, flantUser.UUID)

		clientsData := TestAPI.List(tests.Params{}, url.Values{}).Get("clients").Array()

		Expect(len(clientsData)).To(BeNumerically(">", 0))
		Expect(clientsData[0].Map()).ToNot(HaveKey("origin"))
		clientUUIDS := []string{}
		for i := 0; i < len(clientsData); i++ {
			clientUUIDS = append(clientUUIDS, clientsData[i].Get("uuid").String())
		}
		Expect(clientUUIDS).To(ContainElement(createdClient.UUID))
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomClientCreatePayload()
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID
		createPayload["primary_administrators"] = []string{flantUser.UUID}

		params := tests.Params{
			"expectPayload": func(json gjson.Result) {
				clientData := json.Get("client")
				Expect(clientData.Map()).To(HaveKey("uuid"))
				Expect(clientData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			createdClient := specs.CreateRandomClient(TestAPI, flantUser.UUID)

			TestAPI.Delete(tests.Params{
				"client": createdClient.UUID,
			}, nil)

			TestAPI.Delete(tests.Params{
				"client": createdClient.UUID, "expectStatus": tests.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			createdClient := specs.CreateRandomClient(TestAPI, flantUser.UUID)

			TestAPI.Delete(tests.Params{
				"client": createdClient.UUID,
			}, nil)

			updatePayload := fixtures.RandomClientCreatePayload()
			updatePayload["resource_version"] = createdClient.Version
			TestAPI.Update(tests.Params{
				"client":       createdClient.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func checkPrimaryAdminsStaffExists(flantFlowCfg *config.FlantFlowConfig, clientUUID ext_model.ClientUUID, clientIdentifier string, flantUserUUID model.UserUUID, needExist bool) {
	rbs := RoleBindingAPI.List(tests.Params{
		"tenant": clientUUID,
	}, url.Values{}).Get("role_bindings").Array()
	var adminRoleBinding gjson.Result
	adminRoleBindingExist := false
	for _, rb := range rbs {
		Expect(rb.Map()).To(HaveKey("description"))
		if rb.Get("description").String() == "autocreated rolebinding for primary administrators" {
			adminRoleBinding = rb
			adminRoleBindingExist = true
		}
	}
	if needExist {
		Expect(adminRoleBindingExist).To(BeTrue(), fmt.Sprintf("should exists rolebinding for primary admin"+
			"collected role_bindings:\n %#v", rbs))
		Expect(adminRoleBinding.Get("members").Array()).To(HaveLen(1),
			fmt.Sprintf("Should exists at least 1 user at rolebinding, got rb:\n %s", adminRoleBinding.String()))
		Expect(adminRoleBinding.Get("members").Array()[0].Get("uuid").String()).To(Equal(flantUserUUID),
			fmt.Sprintf("should exists rolebinding with user %s as member, collected role_binding:\n %#v", flantUserUUID, adminRoleBinding))
		Expect(adminRoleBinding.Get("roles").Array()).To(HaveLen(1),
			fmt.Sprintf("Should exists at least 1 role at rolebinding, got rb:\n %s", adminRoleBinding.String()))
		Expect(adminRoleBinding.Get("roles").Array()[0].Get("name").String()).To(Equal(flantFlowCfg.ClientPrimaryAdministratorsRoles[0]),
			fmt.Sprintf("should exists rolebinding with role %s, collected role_binding:\n %#v", flantFlowCfg.ClientPrimaryAdministratorsRoles[0], adminRoleBinding))
	} else {
		Expect(adminRoleBindingExist).To(BeFalse(), fmt.Sprintf("should NOT exists rolebinding for primary admin"+
			"collected role_bindings:\n %#v", rbs))
	}

	// check group exists
	groups := GroupAPI.List(tests.Params{
		"tenant": flantFlowCfg.FlantTenantUUID,
	}, url.Values{}).Get("groups").Array()
	sharedGroupUUID := ""
	for _, group := range groups {
		Expect(group.Map()).To(HaveKey("identifier"))
		if group.Get("identifier").String() == "shared_to_"+clientIdentifier {
			Expect(group.Get("members").Array()[0].Get("uuid").String()).To(Equal(flantUserUUID))
			sharedGroupUUID = group.Get("uuid").String()
		}
	}
	Expect(sharedGroupUUID).ToNot(Equal(""))

	// check IdentitySharing exists
	resp := IdentitySharingAPI.List(tests.Params{
		"tenant": flantFlowCfg.FlantTenantUUID,
	}, url.Values{})
	Expect(resp.Map()).To(HaveKey("identity_sharings"))
	identitySharingExists := false
	for _, is := range resp.Get("identity_sharings").Array() {
		if is.Get("destination_tenant_uuid").String() == clientUUID &&
			len(is.Get("groups").Array()) == 1 && is.Get("groups").Array()[0].Get("uuid").String() == sharedGroupUUID {
			identitySharingExists = true
		}
	}
	if needExist {
		Expect(identitySharingExists).To(BeTrue(), fmt.Sprintf("should exists identitySharing for group "+
			"[%s] from flant [%s] to new client [%s], collected identity_sharings:\n %s", sharedGroupUUID,
			flantFlowCfg.FlantTenantUUID, clientUUID, resp.Get("identity_sharings").String()))
	} else {
		Expect(identitySharingExists).To(BeFalse(), fmt.Sprintf("should NOT exists identitySharing for group "+
			"[%s] from flant [%s] to new client [%s], collected identity_sharings:\n %s", sharedGroupUUID,
			flantFlowCfg.FlantTenantUUID, clientUUID, resp.Get("identity_sharings").String()))
	}
}

func checkIdentitySharingAllFlantGroupExists(flantFlowCfg *config.FlantFlowConfig, clientUUID string, needExist bool) {
	resp := IdentitySharingAPI.List(tests.Params{
		"tenant": flantFlowCfg.FlantTenantUUID,
	}, url.Values{})
	Expect(resp.Map()).To(HaveKey("identity_sharings"))
	identitySharingExists := false
	for _, is := range resp.Get("identity_sharings").Array() {
		if is.Get("destination_tenant_uuid").String() == clientUUID &&
			len(is.Get("groups").Array()) == 1 && is.Get("groups").Array()[0].Get("uuid").String() == flantFlowCfg.AllFlantGroupUUID {
			identitySharingExists = true
		}
	}
	if needExist {
		Expect(identitySharingExists).To(BeTrue(), fmt.Sprintf("should exists identitySharing for group "+
			"flant-all [%s] from flant [%s] to new client [%s], collected identity_sharings:\n %s", flantFlowCfg.AllFlantGroupUUID,
			flantFlowCfg.FlantTenantUUID, clientUUID, resp.Get("identity_sharings").String()))
	} else {
		Expect(identitySharingExists).To(BeFalse(), fmt.Sprintf("should NOT exists identitySharing for group "+
			"flant-all [%s] from flant [%s] to new client [%s], collected identity_sharings:\n %s", flantFlowCfg.AllFlantGroupUUID,
			flantFlowCfg.FlantTenantUUID, clientUUID, resp.Get("identity_sharings").String()))
	}
}

func tryCreateRandomClientWithIdentifier(identifier interface{}, primaryAdminUUID string, statusCodeCondition string) {
	payload := fixtures.RandomClientCreatePayload()
	payload["identifier"] = identifier
	payload["primary_administrators"] = []string{primaryAdminUUID}

	params := tests.Params{
		"expectStatus": tests.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}
