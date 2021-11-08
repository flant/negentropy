package team

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var TestAPI api.TestAPI

var _ = Describe("Team", func() {
	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				payload := fixtures.RandomTeamCreatePayload()
				payload["identifier"] = identifier

				params := api.Params{"expectStatus": api.ExpectStatus(statusCodeCondition)}

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
		createPayload := fixtures.RandomTeamCreatePayload()

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				teamData := json.Get("team")
				Expect(teamData.Map()).To(HaveKey("uuid"))
				Expect(teamData.Map()).To(HaveKey("identifier"))
				Expect(teamData.Get("identifier").String()).To(Equal(createPayload["identifier"].(string)))

				Expect(teamData.Map()).To(HaveKey("resource_version"))
				Expect(teamData.Map()).To(HaveKey("team_type"))
				Expect(teamData.Get("team_type").String()).To(Equal(createPayload["team_type"].(string)))

				Expect(teamData.Map()).To(HaveKey("parent_team_uuid"))

				Expect(teamData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(teamData.Get("resource_version").String()).ToNot(HaveLen(10))
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		createPayload := fixtures.RandomTeamCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Read(api.Params{
			"team": createdData.Get("team.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json, "full_restore")
			},
		}, nil)
	})

	It("can be updated", func() {
		createPayload := fixtures.RandomTeamCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		updatePayload := fixtures.RandomTeamCreatePayload()
		updatePayload["resource_version"] = createdData.Get("team.resource_version").String()
		updatePayload["identifier"] = createdData.Get("team.uuid").String()

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"team": createdData.Get("team.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"team": createdData.Get("team.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json, "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		createPayload := fixtures.RandomTeamCreatePayload()

		var createdData gjson.Result
		TestAPI.Create(api.Params{
			"expectPayload": func(json gjson.Result) {
				createdData = json
			},
		}, nil, createPayload)

		TestAPI.Delete(api.Params{
			"team": createdData.Get("team.uuid").String(),
		}, nil)

		deletedTeamData := TestAPI.Read(api.Params{
			"team":         createdData.Get("team.uuid").String(),
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedTeamData.Get("team.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		createPayload := fixtures.RandomTeamCreatePayload()
		TestAPI.Create(api.Params{}, url.Values{}, createPayload)
		TestAPI.List(api.Params{}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomTeamCreatePayload()
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				teamData := json.Get("team")
				Expect(teamData.Map()).To(HaveKey("uuid"))
				Expect(teamData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})
})
