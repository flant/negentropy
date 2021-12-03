package teammate

import (
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
)

var (
	TestAPI   testapi.TestAPI
	TeamAPI   testapi.TestAPI
	TenantAPI testapi.TestAPI
	ConfigAPI testapi.ConfigAPI
)

var _ = Describe("Teammate", func() {
	var (
		team         model.Team
		flantFlowCfg *config.FlantFlowConfig
	)
	BeforeSuite(func() {
		flantFlowCfg = specs.BaseConfigureFlantFlow(TenantAPI, TeamAPI, ConfigAPI)
		fmt.Printf("%#v\n", flantFlowCfg)
		team = specs.CreateRandomTeam(TeamAPI)
	}, 1.0)
	It("can be created", func() {
		createPayload := fixtures.RandomTeammateCreatePayload(team)
		createPayload["team_uuid"] = team.UUID

		params := testapi.Params{
			"expectPayload": func(json gjson.Result) {
				teammateData := json.Get("teammate")
				Expect(teammateData.Map()).To(HaveKey("uuid"))
				Expect(teammateData.Map()).To(HaveKey("identifier"))
				Expect(teammateData.Map()).To(HaveKey("full_identifier"))
				Expect(teammateData.Map()).To(HaveKey("email"))
				Expect(teammateData.Map()).To(HaveKey("origin"))
				Expect(teammateData.Map()).To(HaveKey("team_uuid"))
				Expect(teammateData.Get("team_uuid").String()).To(Equal(createPayload["team_uuid"].(string)))
				Expect(teammateData.Map()).To(HaveKey("role_at_team"))
				Expect(teammateData.Get("role_at_team").String()).To(Equal(createPayload["role_at_team"].(string)))
				Expect(teammateData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(teammateData.Get("resource_version").String()).ToNot(HaveLen(10))
			},
			"team": team.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)
		createdData := iam_specs.ConvertToGJSON(teammate)

		TestAPI.Read(testapi.Params{
			"team":     teammate.TeamUUID,
			"teammate": teammate.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(createdData, json.Get("teammate"), "extensions")
			},
		}, nil)
	})

	It("can be deleted", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)

		TestAPI.Delete(testapi.Params{
			"team":     teammate.TeamUUID,
			"teammate": teammate.UUID,
		}, nil)

		deletedData := TestAPI.Read(testapi.Params{
			"team":         teammate.TeamUUID,
			"teammate":     teammate.UUID,
			"expectStatus": testapi.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("teammate.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)

		TestAPI.List(testapi.Params{
			"team": teammate.TeamUUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("teammates").Array(),
					iam_specs.ConvertToGJSON(teammate), "extensions")
			},
		}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomTeammateCreatePayload(team)
		originalUUID := createPayload["uuid"]
		createPayload["team_uuid"] = team.UUID

		params := testapi.Params{
			"expectPayload": func(json gjson.Result) {
				teammateData := json.Get("teammate")
				Expect(teammateData.Map()).To(HaveKey("uuid"))
				Expect(teammateData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
			"team": team.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})
})
