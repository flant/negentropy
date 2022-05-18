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
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI            tests.TestAPI
	TenantAPI          tests.TestAPI
	RoleAPI            tests.TestAPI
	TeamAPI            tests.TestAPI
	GroupAPI           tests.TestAPI
	ConfigAPI          testapi.ConfigAPI
	IdentitySharingAPI tests.TestAPI
)

var _ = Describe("Client", func() {
	var flantFlowCfg *config.FlantFlowConfig
	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(TenantAPI, RoleAPI, TeamAPI, GroupAPI, ConfigAPI)
		fmt.Printf("%#v\n", flantFlowCfg)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomClientWithIdentifier(identifier, statusCodeCondition)
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

		// Check identity_sharing is created
		checkIdentitySharingExists(flantFlowCfg, clientUUID, true)
	})

	Context("global uniqueness of client Identifier", func() {
		It("Can not be the same Identifier", func() {
			identifier := uuid.New()
			tryCreateRandomClientWithIdentifier(identifier, "%d == 201")
			tryCreateRandomClientWithIdentifier(identifier, "%d >= 400")
		})
	})

	It("can be read", func() {
		createPayload := fixtures.RandomClientCreatePayload()

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
		createPayload := fixtures.RandomClientCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(tests.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Delete(tests.Params{
			"client": createdData.Get("client.uuid").String(),
		}, nil)

		deletedClientData := TestAPI.Read(tests.Params{
			"client":       createdData.Get("client.uuid").String(),
			"expectStatus": tests.ExpectExactStatus(200),
		}, nil)
		Expect(deletedClientData.Get("client.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))

		// Check identity_sharing is deleted
		checkIdentitySharingExists(flantFlowCfg, createdData.Get("client.uuid").String(), false)
	})

	It("can be listed", func() {
		createPayload := fixtures.RandomClientCreatePayload()
		TestAPI.Create(tests.Params{}, url.Values{}, createPayload)
		clientsData := TestAPI.List(tests.Params{}, url.Values{}).Get("clients")
		Expect(len(clientsData.Array())).To(BeNumerically(">", 0))
		Expect(clientsData.Array()[0].Map()).ToNot(HaveKey("origin"))
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomClientCreatePayload()
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

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
			createPayload := fixtures.RandomClientCreatePayload()
			var createdData gjson.Result
			TestAPI.Create(tests.Params{
				"expectPayload": func(json gjson.Result) {
					createdData = json
				},
			}, nil, createPayload)
			TestAPI.Delete(tests.Params{
				"client": createdData.Get("client.uuid").String(),
			}, nil)

			TestAPI.Delete(tests.Params{
				"client": createdData.Get("client.uuid").String(), "expectStatus": tests.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			createPayload := fixtures.RandomClientCreatePayload()
			var createdData gjson.Result
			TestAPI.Create(tests.Params{
				"expectPayload": func(json gjson.Result) {
					createdData = json
				},
			}, nil, createPayload)
			TestAPI.Delete(tests.Params{
				"client": createdData.Get("client.uuid").String(),
			}, nil)

			updatePayload := fixtures.RandomClientCreatePayload()
			updatePayload["resource_version"] = createdData.Get("client.resource_version").String()
			TestAPI.Update(tests.Params{
				"client":       createdData.Get("client.uuid").String(),
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func checkIdentitySharingExists(flantFlowCfg *config.FlantFlowConfig, clientUUID string, needExist bool) {
	resp := IdentitySharingAPI.List(tests.Params{
		"tenant": flantFlowCfg.FlantTenantUUID,
	}, url.Values{})
	Expect(resp.Map()).To(HaveKey("identity_sharings"))
	identitySharingExists := false
	for _, is := range resp.Get("identity_sharings").Array() {
		if is.Get("destination_tenant_uuid").String() == clientUUID &&
			len(is.Get("groups").Array()) == 1 && is.Get("groups").Array()[0].Get("uuid").String() == flantFlowCfg.AllFlantGroup {
			identitySharingExists = true
		}
	}
	if needExist {
		Expect(identitySharingExists).To(BeTrue(), fmt.Sprintf("should exists identitySharing for group "+
			"flant-all [%s] from flant [%s] to new client [%s], collected identity_sharings:\n %s", flantFlowCfg.AllFlantGroup,
			flantFlowCfg.FlantTenantUUID, clientUUID, resp.Get("identity_sharings").String()))
	} else {
		Expect(identitySharingExists).To(BeFalse(), fmt.Sprintf("should NOT exists identitySharing for group "+
			"flant-all [%s] from flant [%s] to new client [%s], collected identity_sharings:\n %s", flantFlowCfg.AllFlantGroup,
			flantFlowCfg.FlantTenantUUID, clientUUID, resp.Get("identity_sharings").String()))
	}
}

func tryCreateRandomClientWithIdentifier(identifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomClientCreatePayload()
	payload["identifier"] = identifier

	params := tests.Params{
		"expectStatus": tests.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}
