package contact

import (
	json2 "encoding/json"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
)

var (
	TestAPI   testapi.TestAPI
	ClientAPI testapi.TestAPI
)

var _ = Describe("Contact", func() {
	var client model.Client
	BeforeSuite(func() {
		client = specs.CreateRandomClient(ClientAPI)
	}, 1.0)
	It("can be created", func() {
		createPayload := fixtures.RandomContactCreatePayload()
		createPayload["client_uuid"] = client.UUID

		params := testapi.Params{
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
		createdData := iam_specs.ConvertToGJSON(contact)

		TestAPI.Read(testapi.Params{
			"client":  contact.TenantUUID,
			"contact": contact.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(createdData, json.Get("contact"), "extensions")
			},
		}, nil)
	})

	It("can be deleted", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)

		TestAPI.Delete(testapi.Params{
			"client":  contact.TenantUUID,
			"contact": contact.UUID,
		}, nil)

		deletedData := TestAPI.Read(testapi.Params{
			"client":       contact.TenantUUID,
			"contact":      contact.UUID,
			"expectStatus": testapi.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("contact.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		contact := specs.CreateRandomContact(TestAPI, client.UUID)

		TestAPI.List(testapi.Params{
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

		params := testapi.Params{
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
