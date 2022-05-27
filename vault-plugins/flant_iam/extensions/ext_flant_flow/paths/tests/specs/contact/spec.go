package contact

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI    tests.TestAPI
	ClientAPI  tests.TestAPI
	ProjectAPI tests.TestAPI
	TenantAPI  tests.TestAPI
	RoleAPI    tests.TestAPI
	TeamAPI    tests.TestAPI
	GroupAPI   tests.TestAPI
	ConfigAPI  testapi.ConfigAPI
	UserAPI    tests.TestAPI
)

var _ = Describe("Contact", func() {
	var client model.Client
	var flantFlowCfg *config.FlantFlowConfig
	var flantUser iam_model.User

	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(TenantAPI, RoleAPI, TeamAPI, GroupAPI, ConfigAPI)
		fmt.Printf("%#v\n", flantFlowCfg)
		flantUser = iam_specs.CreateRandomUser(UserAPI, flantFlowCfg.FlantTenantUUID)
		client = specs.CreateRandomClient(ClientAPI, flantUser.UUID)
		specs.TryCreateProjects(ProjectAPI, client.UUID, fixtures.Projects()...)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomContactAtTenantWithIdentifier(client.UUID, identifier, statusCodeCondition)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomContactCreatePayload()
		createPayload["client_uuid"] = client.UUID

		params := tests.Params{
			"expectPayload": func(j gjson.Result) {
				contactData := j.Get("contact")
				Expect(contactData.Map()).To(HaveKey("uuid"))
				Expect(contactData.Map()).To(HaveKey("identifier"))
				Expect(contactData.Map()).To(HaveKey("full_identifier"))
				Expect(contactData.Map()).To(HaveKey("email"))
				Expect(contactData.Map()).ToNot(HaveKey("origin"))
				Expect(contactData.Get("uuid").String()).To(HaveLen(36))
				Expect(contactData.Get("resource_version").String()).To(HaveLen(36))
				Expect(contactData.Map()).To(HaveKey("credentials"))
				gotCreds := contactData.Get("credentials").Map()
				b, _ := json.Marshal(createPayload["credentials"])
				expectedCreds := gjson.Parse(string(b)).Map()
				Expect(gotCreds).To(Equal(expectedCreds))
				Expect(contactData.Map()).To(HaveKey("language"))
				Expect(contactData.Get("language").String()).To(Equal(createPayload["language"]))
			},
			"client": client.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	Context("tenant uniqueness of contact identifier", func() {
		identifier := uuid.New()
		It("Can be created contact with some identifier", func() {
			tryCreateRandomContactAtTenantWithIdentifier(client.UUID, identifier, "%d == 201")
		})
		It("Can not be the same identifier at the same tenant", func() {
			tryCreateRandomContactAtTenantWithIdentifier(client.UUID, identifier, "%d >= 400")
		})
		It("Can be same identifier at other tenant", func() {
			client = specs.CreateRandomClient(ClientAPI, flantUser.UUID)
			tryCreateRandomContactAtTenantWithIdentifier(client.UUID, identifier, "%d == 201")
		})
	})

	It("can be read", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)
		createdData := iam_specs.ConvertToGJSON(contact)

		TestAPI.Read(tests.Params{
			"client":  contact.TenantUUID,
			"contact": contact.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(createdData, json.Get("contact"), "extensions")
				Expect(json.Get("contact").Map()).ToNot(HaveKey("origin"))
			},
		}, nil)
	})

	It("can be updated", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)
		updatePayload := fixtures.RandomContactCreatePayload()
		updatePayload["uuid"] = contact.UUID
		updatePayload["client_uuid"] = client.UUID
		updatePayload["resource_version"] = contact.Version

		TestAPI.Update(tests.Params{
			"client":       contact.TenantUUID,
			"contact":      contact.UUID,
			"expectStatus": tests.ExpectExactStatus(200),
			"expectPayload": func(json gjson.Result) {
				Expect(json.Get("contact").Map()).ToNot(HaveKey("origin"))
			},
		}, nil, updatePayload)

		TestAPI.Read(tests.Params{
			"client":  contact.TenantUUID,
			"contact": contact.UUID,
			"expectPayload": func(json gjson.Result) {
				contactData := json.Get("contact")
				iam_specs.IsMapSubsetOfSetExceptKeys(updatePayload, contactData, "archiving_timestamp",
					"archiving_hash", "uuid", "resource_version", "origin", "tenant_uuid", "additional_phones",
					"client_uuid", "full_identifier", "additional_emails", "extensions")
				Expect(contactData.Map()).ToNot(HaveKey("origin"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)

		deleteContact(contact)

		deletedData := TestAPI.Read(tests.Params{
			"client":       contact.TenantUUID,
			"contact":      contact.UUID,
			"expectStatus": tests.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("contact.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)

		TestAPI.List(tests.Params{
			"client": contact.TenantUUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("contacts").Array(),
					iam_specs.ConvertToGJSON(contact), "extensions")
				contactsArray := json.Get("contacts").Array()
				Expect(len(contactsArray)).To(BeNumerically(">", 0))
				Expect(contactsArray[0].Map()).ToNot(HaveKey("origin"))
			},
		}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomContactCreatePayload()
		originalUUID := createPayload["uuid"]
		createPayload["client_uuid"] = client.UUID

		params := tests.Params{
			"expectPayload": func(json gjson.Result) {
				contactData := json.Get("contact")
				Expect(contactData.Map()).To(HaveKey("uuid"))
				Expect(contactData.Map()["uuid"].String()).To(Equal(originalUUID))
				Expect(contactData.Map()).ToNot(HaveKey("origin"))
			},
			"client": client.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			contact := specs.CreateRandomContact(TestAPI, client.UUID)
			deleteContact(contact)

			TestAPI.Delete(tests.Params{
				"client":       contact.TenantUUID,
				"contact":      contact.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			contact := specs.CreateRandomContact(TestAPI, client.UUID)
			deleteContact(contact)

			updatePayload := fixtures.RandomContactCreatePayload()
			updatePayload["uuid"] = contact.UUID
			updatePayload["client_uuid"] = client.UUID
			updatePayload["resource_version"] = contact.Version

			TestAPI.Update(tests.Params{
				"client":       contact.TenantUUID,
				"contact":      contact.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})

	Context("restoring deleted contact", func() {
		It("can be restored after deleting", func() {
			contact := specs.CreateRandomContact(TestAPI, client.UUID)
			deleteContact(contact)

			TestAPI.Restore(tests.Params{
				"client":       contact.TenantUUID,
				"contact":      contact.UUID,
				"expectStatus": tests.ExpectExactStatus(200),
				"expectPayload": func(json gjson.Result) {
					contactData := json.Get("contact")
					Expect(contactData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically("==", int64(0))))
				},
			}, nil)
		})

		It("cant be restored after deleting client", func() {
			otherClient := specs.CreateRandomClient(ClientAPI, flantUser.UUID)
			contact := specs.CreateRandomContact(TestAPI, otherClient.UUID)
			deleteContact(contact)
			ClientAPI.Delete(tests.Params{
				"expectStatus": tests.ExpectExactStatus(http.StatusNoContent),
				"client":       otherClient.UUID,
			}, nil)

			TestAPI.Restore(tests.Params{
				"client":       contact.TenantUUID,
				"contact":      contact.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil)

			TestAPI.Read(tests.Params{
				"client":       contact.TenantUUID,
				"contact":      contact.UUID,
				"expectStatus": tests.ExpectExactStatus(200),
				"expectPayload": func(json gjson.Result) {
					contactData := json.Get("contact")
					Expect(contactData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
				},
			}, nil)
		})
	})
})

func tryCreateRandomContactAtTenantWithIdentifier(clientUUID string,
	contactIdentifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomContactCreatePayload()
	payload["identifier"] = contactIdentifier

	params := tests.Params{
		"client":       clientUUID,
		"expectStatus": tests.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}

func deleteContact(contact model.FullContact) {
	TestAPI.Delete(tests.Params{
		"expectStatus": tests.ExpectExactStatus(http.StatusNoContent),
		"client":       contact.TenantUUID,
		"contact":      contact.UUID,
	}, nil)
}
