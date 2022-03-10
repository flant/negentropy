package team

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
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI testapi.TestAPI

	TenantAPI testapi.TestAPI
	RoleAPI   testapi.TestAPI
	ConfigAPI testapi.ConfigAPI

	GroupAPI testapi.TestAPI
)

var _ = Describe("Team", func() {
	var flantFlowCfg *config.FlantFlowConfig
	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(TenantAPI, RoleAPI, TestAPI, ConfigAPI)
		fmt.Printf("%#v\n", flantFlowCfg)
	}, 1.0)
	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				payload := fixtures.RandomTeamCreatePayload()
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
		createPayload := fixtures.RandomTeamCreatePayload()
		var directGroupUUID model.GroupUUID
		params := testapi.Params{
			"expectPayload": func(json gjson.Result) {
				teamData := json.Get("team")
				Expect(teamData.Map()).To(HaveKey("uuid"))
				Expect(teamData.Get("uuid").String()).To(HaveLen(36))
				Expect(teamData.Map()).To(HaveKey("identifier"))
				Expect(teamData.Get("identifier").String()).To(Equal(createPayload["identifier"].(string)))

				Expect(teamData.Map()).To(HaveKey("resource_version"))
				Expect(teamData.Get("resource_version").String()).To(HaveLen(36))
				Expect(teamData.Map()).To(HaveKey("team_type"))
				Expect(teamData.Get("team_type").String()).To(Equal(createPayload["team_type"].(string)))

				Expect(teamData.Map()).To(HaveKey("parent_team_uuid"))

				Expect(teamData.Map()).To(HaveKey("groups"))
				Expect(teamData.Get("groups").Array()).To(HaveLen(1))
				directLinkedGroup := teamData.Get("groups").Array()[0]
				Expect(directLinkedGroup.Map()).To(HaveKey("type"))
				Expect(directLinkedGroup.Get("type").String()).To(Equal(usecase.DirectMembersGroupType))
				Expect(directLinkedGroup.Map()).To(HaveKey("uuid"))
				directGroupUUID = directLinkedGroup.Get("uuid").String()
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)
		GroupAPI.Read(testapi.Params{
			"tenant": flantFlowCfg.FlantTenantUUID,
			"group":  directGroupUUID,
			"expectPayload": func(json gjson.Result) {
				groupData := json.Get("group")
				Expect(groupData.Map()).To(HaveKey("uuid"))
			},
		},
			nil)
	})

	It("can be read", func() {
		team := specs.CreateRandomTeam(TestAPI)

		TestAPI.Read(testapi.Params{
			"team": team.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(iam_specs.ConvertToGJSON(team), json.Get("team"), "full_restore")
			},
		}, nil)
	})

	It("can be updated", func() {
		team := specs.CreateRandomTeam(TestAPI)

		updatePayload := fixtures.RandomTeamCreatePayload()
		updatePayload["resource_version"] = team.Version
		updatePayload["team_type"] = team.TeamType

		updateData := TestAPI.Update(testapi.Params{
			"team": team.UUID,
		}, nil, updatePayload)

		TestAPI.Read(testapi.Params{
			"team": team.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(updateData.Get("team"), json.Get("team"), "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		team := specs.CreateRandomTeam(TestAPI)

		TestAPI.Delete(testapi.Params{
			"team": team.UUID,
		}, nil)

		deletedTeamData := TestAPI.Read(testapi.Params{
			"team":         team.UUID,
			"expectStatus": testapi.ExpectExactStatus(200),
		}, nil).Get("team")
		Expect(deletedTeamData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
		Expect(deletedTeamData.Map()).To(HaveKey("groups"))
		Expect(deletedTeamData.Get("groups").Array()).To(HaveLen(0))
	})

	It("can be listed", func() {
		team := specs.CreateRandomTeam(TestAPI)

		TestAPI.List(testapi.Params{"expectPayload": func(json gjson.Result) {
			iam_specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("teams").Array(),
				iam_specs.ConvertToGJSON(team))
		}}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomTeamCreatePayload()
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := testapi.Params{
			"expectPayload": func(json gjson.Result) {
				teamData := json.Get("team")
				Expect(teamData.Map()).To(HaveKey("uuid"))
				Expect(teamData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	It("can't be deleted L1Team", func() {
		TestAPI.Delete(testapi.Params{
			"team":         flantFlowCfg.SpecificTeams[config.L1],
			"expectStatus": testapi.ExpectExactStatus(400),
		}, nil)
	})

	It("can't be deleted OkMeterTeam", func() {
		TestAPI.Delete(testapi.Params{
			"team":         flantFlowCfg.SpecificTeams[config.Okmeter],
			"expectStatus": testapi.ExpectExactStatus(400),
		}, nil)
	})

	It("can't be deleted Mk8sTeam", func() {
		TestAPI.Delete(testapi.Params{
			"team":         flantFlowCfg.SpecificTeams[config.Mk8s],
			"expectStatus": testapi.ExpectExactStatus(400),
		}, nil)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			team := specs.CreateRandomTeam(TestAPI)
			TestAPI.Delete(testapi.Params{
				"team": team.UUID,
			}, nil)

			TestAPI.Delete(testapi.Params{
				"team":         team.UUID,
				"expectStatus": testapi.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			team := specs.CreateRandomTeam(TestAPI)
			TestAPI.Delete(testapi.Params{
				"team": team.UUID,
			}, nil)

			updatePayload := fixtures.RandomTeamCreatePayload()
			updatePayload["resource_version"] = team.Version
			updatePayload["team_type"] = team.TeamType

			TestAPI.Update(testapi.Params{
				"team":         team.UUID,
				"expectStatus": testapi.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})
