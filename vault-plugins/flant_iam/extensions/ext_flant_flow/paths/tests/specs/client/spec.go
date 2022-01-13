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
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI   testapi.TestAPI
	TenantAPI testapi.TestAPI
	RoleAPI   testapi.TestAPI
	TeamAPI   testapi.TestAPI
	ConfigAPI testapi.ConfigAPI
)

var _ = Describe("Client", func() {
	var flantFlowCfg *config.FlantFlowConfig
	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(TenantAPI, RoleAPI, TeamAPI, ConfigAPI)
		fmt.Printf("%#v\n", flantFlowCfg)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				payload := fixtures.RandomClientCreatePayload()
				payload["identifier"] = identifier

				params := testapi.Params{"expectStatus": testapi.ExpectStatus(statusCodeCondition)}

				TestAPI.Create(params, nil, payload)
			},
			Entry("number allowed", 100, "%d == 201"),
			Entry("absent identifier forbidden", nil, "%d >= 400"),
			Entry("empty string forbidden", "", "%d >= 400"),
			Entry("array forbidden", []string{"a"}, "%d >= 400"),
			Entry("object forbidden", map[string]int{"a": 1}, "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomClientCreatePayload()

		params := testapi.Params{
			"expectPayload": func(json gjson.Result) {
				clientData := json.Get("client")
				Expect(clientData.Map()).To(HaveKey("uuid"))
				Expect(clientData.Map()).To(HaveKey("identifier"))
				Expect(clientData.Map()).To(HaveKey("resource_version"))
				Expect(clientData.Get("uuid").String()).To(HaveLen(36))
				Expect(clientData.Get("resource_version").String()).To(HaveLen(36))
				Expect(clientData.Map()).To(HaveKey("origin"))
				Expect(clientData.Get("origin").String()).To(Equal(string(consts.OriginFlantFlow)))
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		createPayload := fixtures.RandomClientCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(testapi.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Read(testapi.Params{
			"client": createdData.Get("client.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(createdData, json, "full_restore")
			},
		}, nil)
	})

	It("can be updated", func() {
		createPayload := fixtures.RandomClientCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(testapi.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		updatePayload := fixtures.RandomClientCreatePayload()
		updatePayload["resource_version"] = createdData.Get("client.resource_version").String()

		var updateData gjson.Result
		TestAPI.Update(testapi.Params{
			"client": createdData.Get("client.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(testapi.Params{
			"client": createdData.Get("client.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(updateData, json, "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		createPayload := fixtures.RandomClientCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(testapi.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Delete(testapi.Params{
			"client": createdData.Get("client.uuid").String(),
		}, nil)

		deletedClientData := TestAPI.Read(testapi.Params{
			"client":       createdData.Get("client.uuid").String(),
			"expectStatus": testapi.ExpectExactStatus(200),
		}, nil)
		Expect(deletedClientData.Get("client.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		createPayload := fixtures.RandomClientCreatePayload()
		TestAPI.Create(testapi.Params{}, url.Values{}, createPayload)
		TestAPI.List(testapi.Params{}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomClientCreatePayload()
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := testapi.Params{
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
			TestAPI.Create(testapi.Params{
				"expectPayload": func(json gjson.Result) {
					createdData = json
				},
			}, nil, createPayload)
			TestAPI.Delete(testapi.Params{
				"client": createdData.Get("client.uuid").String(),
			}, nil)

			TestAPI.Delete(testapi.Params{
				"client": createdData.Get("client.uuid").String(), "expectStatus": testapi.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			createPayload := fixtures.RandomClientCreatePayload()
			var createdData gjson.Result
			TestAPI.Create(testapi.Params{
				"expectPayload": func(json gjson.Result) {
					createdData = json
				},
			}, nil, createPayload)
			TestAPI.Delete(testapi.Params{
				"client": createdData.Get("client.uuid").String(),
			}, nil)

			updatePayload := fixtures.RandomClientCreatePayload()
			updatePayload["resource_version"] = createdData.Get("client.resource_version").String()
			TestAPI.Update(testapi.Params{
				"client":       createdData.Get("client.uuid").String(),
				"expectStatus": testapi.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})
