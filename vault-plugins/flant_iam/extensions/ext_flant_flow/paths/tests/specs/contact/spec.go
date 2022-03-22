package contact

import (
	"encoding/json"
	"fmt"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

var (
	TestAPI    tests.TestAPI
	ClientAPI  tests.TestAPI
	ProjectAPI tests.TestAPI
	TenantAPI  tests.TestAPI
	RoleAPI    tests.TestAPI
	TeamAPI    tests.TestAPI
	ConfigAPI  testapi.ConfigAPI
)

var _ = Describe("Contact", func() {
	var client model.Client
	var flantFlowCfg *config.FlantFlowConfig
	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(TenantAPI, RoleAPI, TeamAPI, ConfigAPI)
		fmt.Printf("%#v\n", flantFlowCfg)
		client = specs.CreateRandomClient(ClientAPI)
		specs.TryCreateProjects(ProjectAPI, client.UUID, fixtures.Projects()...)
	}, 1.0)
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
				Expect(contactData.Map()).To(HaveKey("origin"))
				Expect(contactData.Get("uuid").String()).To(HaveLen(36))
				Expect(contactData.Get("resource_version").String()).To(HaveLen(36))
				Expect(contactData.Map()).To(HaveKey("credentials"))
				gotCreds := contactData.Get("credentials").Map()
				b, _ := json.Marshal(createPayload["credentials"])
				expectedCreds := gjson.Parse(string(b)).Map()
				Expect(gotCreds).To(Equal(expectedCreds))
				Expect(contactData.Map()).To(HaveKey("origin"))
				Expect(contactData.Get("origin").String()).To(Equal(string(consts.OriginFlantFlow)))
			},
			"client": client.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)
		createdData := iam_specs.ConvertToGJSON(contact)

		TestAPI.Read(tests.Params{
			"client":  contact.TenantUUID,
			"contact": contact.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(createdData, json.Get("contact"), "extensions")
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
		}, nil, updatePayload)
	})

	It("can be deleted", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)

		TestAPI.Delete(tests.Params{
			"client":  contact.TenantUUID,
			"contact": contact.UUID,
		}, nil)

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
			},
			"client": client.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			contact := specs.CreateRandomContact(TestAPI, client.UUID)
			TestAPI.Delete(tests.Params{
				"client":  contact.TenantUUID,
				"contact": contact.UUID,
			}, nil)

			TestAPI.Delete(tests.Params{
				"client":       contact.TenantUUID,
				"contact":      contact.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			contact := specs.CreateRandomContact(TestAPI, client.UUID)
			TestAPI.Delete(tests.Params{
				"client":  contact.TenantUUID,
				"contact": contact.UUID,
			}, nil)

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
})
