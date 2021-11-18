package contact

import (
	json2 "encoding/json"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
)

var (
	TestAPI   api.TestAPI
	ClientAPI api.TestAPI
)

var _ = Describe("Contact", func() {
	var client model.Client
	BeforeSuite(func() {
		client = specs.CreateRandomClient(ClientAPI)
	}, 1.0)
	It("can be created", func() {
		createPayload := fixtures.RandomContactCreatePayload()
		createPayload["client_uuid"] = client.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				contactData := json.Get("contact")
				Expect(contactData.Map()).To(HaveKey("uuid"))
				Expect(contactData.Map()).To(HaveKey("identifier"))
				Expect(contactData.Map()).To(HaveKey("full_identifier"))
				Expect(contactData.Map()).To(HaveKey("email"))
				Expect(contactData.Map()).To(HaveKey("origin"))
				Expect(contactData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(contactData.Get("resource_version").String()).ToNot(HaveLen(10))
				Expect(contactData.Map()).To(HaveKey("credentials"))
				gotCreds := contactData.Get("credentials").Map()
				b, _ := json2.Marshal(createPayload["credentials"])
				expectedCreds := gjson.Parse(string(b)).Map()
				Expect(gotCreds).To(Equal(expectedCreds))
			},
			"client": client.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)
		createdData := specs.ConvertToGJSON(contact)

		TestAPI.Read(api.Params{
			"client":  contact.TenantUUID,
			"contact": contact.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("contact"), "extensions")
			},
		}, nil)
	})

	It("can be deleted", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)

		TestAPI.Delete(api.Params{
			"client":  contact.TenantUUID,
			"contact": contact.UUID,
		}, nil)

		deletedData := TestAPI.Read(api.Params{
			"client":       contact.TenantUUID,
			"contact":      contact.UUID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("contact.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)

		TestAPI.List(api.Params{
			"client": contact.TenantUUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("contacts").Array(),
					specs.ConvertToGJSON(contact), "extensions")
			},
		}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomContactCreatePayload()
		originalUUID := createPayload["uuid"]
		createPayload["client_uuid"] = client.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				contactData := json.Get("contact")
				Expect(contactData.Map()).To(HaveKey("uuid"))
				Expect(contactData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
			"client": client.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})
})
