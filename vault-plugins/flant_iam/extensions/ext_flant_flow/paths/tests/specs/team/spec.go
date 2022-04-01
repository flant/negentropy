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
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI tests.TestAPI

	TenantAPI tests.TestAPI
	RoleAPI   tests.TestAPI
	ConfigAPI testapi.ConfigAPI

	GroupAPI tests.TestAPI
)

var _ = Describe("Team", func() {
	var flantFlowCfg *config.FlantFlowConfig
	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(TenantAPI, RoleAPI, TestAPI, GroupAPI, ConfigAPI)
		fmt.Printf("%#v\n", flantFlowCfg)
	}, 1.0)
	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomTeamWithIdentifier(identifier, statusCodeCondition)
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
		createPayload := fixtures.RandomTeamCreatePayload()
		var directGroupUUID model.GroupUUID
		params := tests.Params{
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
		GroupAPI.Read(tests.Params{
			"tenant": flantFlowCfg.FlantTenantUUID,
			"group":  directGroupUUID,
			"expectPayload": func(json gjson.Result) {
				groupData := json.Get("group")
				Expect(groupData.Map()).To(HaveKey("uuid"))
			},
		},
			nil)
	})

	Context("global uniqueness of team Identifier", func() {
		It("Can not be the same Identifier", func() {
			identifier := uuid.New()
			tryCreateRandomTeamWithIdentifier(identifier, "%d == 201")
			tryCreateRandomTeamWithIdentifier(identifier, "%d >= 400")
		})
	})

	It("can be read", func() {
		team := specs.CreateRandomTeam(TestAPI)

		TestAPI.Read(tests.Params{
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

		updateData := TestAPI.Update(tests.Params{
			"team": team.UUID,
		}, nil, updatePayload)

		TestAPI.Read(tests.Params{
			"team": team.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(updateData.Get("team"), json.Get("team"), "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		team := specs.CreateRandomTeam(TestAPI)

		TestAPI.Delete(tests.Params{
			"team": team.UUID,
		}, nil)

		deletedTeamData := TestAPI.Read(tests.Params{
			"team":         team.UUID,
			"expectStatus": tests.ExpectExactStatus(200),
		}, nil).Get("team")
		Expect(deletedTeamData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
		Expect(deletedTeamData.Map()).To(HaveKey("groups"))
		Expect(deletedTeamData.Get("groups").Array()).To(HaveLen(0))
	})

	It("can be listed", func() {
		team := specs.CreateRandomTeam(TestAPI)

		TestAPI.List(tests.Params{"expectPayload": func(json gjson.Result) {
			iam_specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("teams").Array(),
				iam_specs.ConvertToGJSON(team))
		}}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomTeamCreatePayload()
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := tests.Params{
			"expectPayload": func(json gjson.Result) {
				teamData := json.Get("team")
				Expect(teamData.Map()).To(HaveKey("uuid"))
				Expect(teamData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	It("can't be deleted L1Team", func() {
		TestAPI.Delete(tests.Params{
			"team":         flantFlowCfg.SpecificTeams[config.L1],
			"expectStatus": tests.ExpectExactStatus(400),
		}, nil)
	})

	It("can't be deleted OkMeterTeam", func() {
		TestAPI.Delete(tests.Params{
			"team":         flantFlowCfg.SpecificTeams[config.Okmeter],
			"expectStatus": tests.ExpectExactStatus(400),
		}, nil)
	})

	It("can't be deleted Mk8sTeam", func() {
		TestAPI.Delete(tests.Params{
			"team":         flantFlowCfg.SpecificTeams[config.Mk8s],
			"expectStatus": tests.ExpectExactStatus(400),
		}, nil)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			team := specs.CreateRandomTeam(TestAPI)
			TestAPI.Delete(tests.Params{
				"team": team.UUID,
			}, nil)

			TestAPI.Delete(tests.Params{
				"team":         team.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			team := specs.CreateRandomTeam(TestAPI)
			TestAPI.Delete(tests.Params{
				"team": team.UUID,
			}, nil)

			updatePayload := fixtures.RandomTeamCreatePayload()
			updatePayload["resource_version"] = team.Version
			updatePayload["team_type"] = team.TeamType

			TestAPI.Update(tests.Params{
				"team":         team.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func tryCreateRandomTeamWithIdentifier(identifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomTeamCreatePayload()
	payload["identifier"] = identifier

	params := tests.Params{
		"expectStatus": tests.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}
