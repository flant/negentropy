package teammate

import (
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
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

var (
	TestAPI   testapi.TestAPI
	TeamAPI   testapi.TestAPI
	RoleAPI   testapi.TestAPI
	TenantAPI testapi.TestAPI
	ConfigAPI testapi.ConfigAPI

	GroupAPI testapi.TestAPI
)

var _ = Describe("Teammate", func() {
	var team model.Team
	var cfg *config.FlantFlowConfig

	BeforeSuite(func() {
		cfg = specs.BaseConfigureFlantFlow(TenantAPI, RoleAPI, ConfigAPI)
		team = specs.CreateRandomTeam(TeamAPI)
	}, 1.0)
	It("can be created", func() {
		createPayload := fixtures.RandomTeammateCreatePayload(team)
		createPayload["team_uuid"] = team.UUID
		var teammateUUID model2.UserUUID

		params := testapi.Params{
			"expectPayload": func(json gjson.Result) {
				teammateData := json.Get("teammate")
				Expect(teammateData.Map()).To(HaveKey("uuid"))
				teammateUUID = teammateData.Get("uuid").String()
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
		checkTeamOfTeammateHasGroupsWithTeammate(cfg.FlantTenantUUID, team.UUID, teammateUUID, true)
	})

	It("can be read", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)

		TestAPI.Read(testapi.Params{
			"team":     teammate.TeamUUID,
			"teammate": teammate.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(iam_specs.ConvertToGJSON(teammate), json.Get("teammate"), "extensions")
			},
		}, nil)
	})

	It("can be updated", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)
		updatePayload := fixtures.RandomTeamCreatePayload()
		delete(updatePayload, "uuid")
		delete(updatePayload, "team_uuid")
		updatePayload["resource_version"] = teammate.Version
		updatePayload["role_at_team"] = teammate.RoleAtTeam
		updateData := TestAPI.Update(testapi.Params{
			"team":     teammate.TeamUUID,
			"teammate": teammate.UUID,
		}, nil, updatePayload)

		Expect(updateData.Get("teammate.identifier").String()).To(Equal(updatePayload["identifier"]))
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
		checkTeamOfTeammateHasGroupsWithTeammate(cfg.FlantTenantUUID, team.UUID, teammate.UUID, false)
	})

	It("can be listed", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)

		TestAPI.List(testapi.Params{
			"team": teammate.TeamUUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("teammates").Array(),
					iam_specs.ConvertToGJSON(teammate), "extensions") // server_access extension has map inside, so no guarantees to equity
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

	Context("after deletion", func() {
		It("can't be deleted", func() {
			teammate := specs.CreateRandomTeammate(TestAPI, team)
			TestAPI.Delete(testapi.Params{
				"team":     teammate.TeamUUID,
				"teammate": teammate.UUID,
			}, nil)

			TestAPI.Delete(testapi.Params{
				"team":         teammate.TeamUUID,
				"teammate":     teammate.UUID,
				"expectStatus": testapi.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			teammate := specs.CreateRandomTeammate(TestAPI, team)
			TestAPI.Delete(testapi.Params{
				"team":     teammate.TeamUUID,
				"teammate": teammate.UUID,
			}, nil)

			updatePayload := fixtures.RandomTeamCreatePayload()
			delete(updatePayload, "uuid")
			delete(updatePayload, "team_uuid")
			updatePayload["resource_version"] = teammate.Version
			updatePayload["role_at_team"] = teammate.RoleAtTeam
			TestAPI.Update(testapi.Params{
				"team":         teammate.TeamUUID,
				"teammate":     teammate.UUID,
				"expectStatus": testapi.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func checkTeamOfTeammateHasGroupsWithTeammate(flantTenantUUID model2.TenantUUID, teamUUID model.TeamUUID,
	teammateUUID model2.UserUUID, expectHas bool) {
	respData := TeamAPI.Read(testapi.Params{
		"team": teamUUID,
	}, nil)
	Expect(respData.Map()).To(HaveKey("team"))
	teamData := respData.Get("team")
	Expect(teamData.Map()).To(HaveKey("groups"))
	Expect(teamData.Get("groups").Array()).To(HaveLen(1))
	directLinkedGroup := teamData.Get("groups").Array()[0]
	Expect(directLinkedGroup.Map()).To(HaveKey("type"))
	Expect(directLinkedGroup.Get("type").String()).To(Equal(usecase.DirectMembersGroupType))
	Expect(directLinkedGroup.Map()).To(HaveKey("uuid"))
	directGroupUUID := directLinkedGroup.Get("uuid").String()
	specs.CheckGroupHasUser(GroupAPI, flantTenantUUID, directGroupUUID, teammateUUID, expectHas)
}
