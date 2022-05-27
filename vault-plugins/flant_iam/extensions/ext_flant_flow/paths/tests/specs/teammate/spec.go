package teammate

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
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI   tests.TestAPI
	TeamAPI   tests.TestAPI
	RoleAPI   tests.TestAPI
	ConfigAPI testapi.ConfigAPI

	GroupAPI tests.TestAPI
)

var _ = Describe("Teammate", func() {
	var team model.Team
	var cfg *config.FlantFlowConfig

	BeforeSuite(func() {
		cfg = specs.BaseConfigureFlantFlow(RoleAPI, ConfigAPI)
		team = specs.CreateRandomTeam(TeamAPI)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomTeammateAtTeamWithIdentifier(team, identifier, statusCodeCondition)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomTeammateCreatePayload(team)
		var teammateUUID model2.UserUUID

		params := tests.Params{
			"expectPayload": func(json gjson.Result) {
				teammateData := json.Get("teammate")
				Expect(teammateData.Map()).To(HaveKey("uuid"))
				teammateUUID = teammateData.Get("uuid").String()
				Expect(teammateData.Map()).To(HaveKey("identifier"))
				Expect(teammateData.Map()).To(HaveKey("full_identifier"))
				Expect(teammateData.Map()).To(HaveKey("email"))
				Expect(teammateData.Map()).To(HaveKey("team_uuid"))
				Expect(teammateData.Get("team_uuid").String()).To(Equal(createPayload["team_uuid"].(string)))
				Expect(teammateData.Map()).To(HaveKey("role_at_team"))
				Expect(teammateData.Get("role_at_team").String()).To(Equal(createPayload["role_at_team"].(string)))
				Expect(teammateData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(teammateData.Get("resource_version").String()).ToNot(HaveLen(10))
				Expect(teammateData.Map()).ToNot(HaveKey("origin"))
				Expect(teammateData.Map()).To(HaveKey("language"))
				Expect(teammateData.Get("language").String()).To(Equal(createPayload["language"]))
				Expect(teammateData.Map()).To(HaveKey("gitlab.com"))
				Expect(teammateData.Get(`gitlab\.com`).String()).To(Equal(createPayload["gitlab.com"]))
				Expect(teammateData.Map()).To(HaveKey("github.com"))
				Expect(teammateData.Get(`github\.com`).String()).To(Equal(createPayload["github.com"]))
				Expect(teammateData.Map()).To(HaveKey("telegram"))
				Expect(teammateData.Get(`telegram`).String()).To(Equal(createPayload["telegram"]))
				Expect(teammateData.Map()).To(HaveKey("habr.com"))
				Expect(teammateData.Get(`habr\.com`).String()).To(Equal(createPayload["habr.com"]))
			},
			"team": team.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
		checkTeamHasGroupsWithTeammate(cfg.FlantTenantUUID, team.UUID, teammateUUID, true, usecase.DirectMembersGroupType)
		checkTeammateInAllFlantGroup(cfg, teammateUUID, true)
	})

	Context("global uniqueness of teammate identifier", func() {
		identifier := uuid.New()
		It("Can be created teammate with some identifier", func() {
			tryCreateRandomTeammateAtTeamWithIdentifier(team, identifier, "%d == 201")
		})
		It("Can not be the same identifier at the same team", func() {
			tryCreateRandomTeammateAtTeamWithIdentifier(team, identifier, "%d >= 400")
		})
		It("Can not be same identifier at other team", func() {
			team = specs.CreateRandomTeam(TeamAPI)
			tryCreateRandomTeammateAtTeamWithIdentifier(team, identifier, "%d >= 400")
		})
	})

	It("can be read", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)

		TestAPI.Read(tests.Params{
			"team":     teammate.TeamUUID,
			"teammate": teammate.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(iam_specs.ConvertToGJSON(teammate), json.Get("teammate"), "extensions")
				Expect(json.Get("teammate").Map()).ToNot(HaveKey("origin"))
			},
		}, nil)
	})

	It("can be updated", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)
		updatePayload := fixtures.RandomTeammateCreatePayload(team)
		delete(updatePayload, "uuid")
		delete(updatePayload, "team_uuid")
		updatePayload["resource_version"] = teammate.Version
		updatePayload["role_at_team"] = teammate.RoleAtTeam
		updateData := TestAPI.Update(tests.Params{
			"team":     teammate.TeamUUID,
			"teammate": teammate.UUID,
			"expectPayload": func(json gjson.Result) {
				Expect(json.Get("teammate").Map()).ToNot(HaveKey("origin"))
			},
		}, nil, updatePayload)

		Expect(updateData.Get("teammate.identifier").String()).To(Equal(updatePayload["identifier"]))
	})

	It("can be deleted", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)

		TestAPI.Delete(tests.Params{
			"team":     teammate.TeamUUID,
			"teammate": teammate.UUID,
		}, nil)

		deletedData := TestAPI.Read(tests.Params{
			"team":         teammate.TeamUUID,
			"teammate":     teammate.UUID,
			"expectStatus": tests.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("teammate.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
		checkTeamHasGroupsWithTeammate(cfg.FlantTenantUUID, team.UUID, teammate.UUID, false, usecase.DirectMembersGroupType, usecase.DirectManagersGroupType)
		checkTeammateInAllFlantGroup(cfg, teammate.UUID, false)
	})

	It("can be listed", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)

		TestAPI.List(tests.Params{
			"team": teammate.TeamUUID,
			"expectPayload": func(json gjson.Result) {
				teammatesArray := json.Get("teammates").Array()
				iam_specs.CheckArrayContainsElementByUUIDExceptKeys(teammatesArray,
					iam_specs.ConvertToGJSON(teammate), "extensions") // server_access extension has map inside, so no guarantees to equity
				Expect(len(teammatesArray)).To(BeNumerically(">", 0))
				Expect(teammatesArray[0].Map()).ToNot(HaveKey("origin"))
			},
		}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomTeammateCreatePayload(team)
		originalUUID := createPayload["uuid"]
		createPayload["team_uuid"] = team.UUID

		params := tests.Params{
			"expectPayload": func(json gjson.Result) {
				teammateData := json.Get("teammate")
				Expect(teammateData.Map()).To(HaveKey("uuid"))
				Expect(teammateData.Map()["uuid"].String()).To(Equal(originalUUID))
				Expect(teammateData.Map()).ToNot(HaveKey("origin"))
			},
			"team": team.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	It("team can be changed", func() {
		teammate := specs.CreateRandomTeammate(TestAPI, team)
		newTeam := specs.CreateRandomTeamWithSpecificType(TeamAPI, team.TeamType)
		updatePayload := map[string]interface{}{
			"resource_version": teammate.Version,
			"role_at_team":     teammate.RoleAtTeam,
			"identifier":       teammate.Identifier,
			"new_team_uuid":    newTeam.UUID,
		}
		updateData := TestAPI.Update(tests.Params{
			"team":     teammate.TeamUUID,
			"teammate": teammate.UUID,
		}, nil, updatePayload)

		readData := TestAPI.Read(tests.Params{
			"team":     newTeam.UUID,
			"teammate": teammate.UUID,
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(updateData.Get("teammate"), json.Get("teammate"), "extensions")
			},
		}, nil)

		Expect(readData.Get("teammate.team_uuid").String()).To(Equal(newTeam.UUID))
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			teammate := specs.CreateRandomTeammate(TestAPI, team)
			TestAPI.Delete(tests.Params{
				"team":     teammate.TeamUUID,
				"teammate": teammate.UUID,
			}, nil)

			TestAPI.Delete(tests.Params{
				"team":         teammate.TeamUUID,
				"teammate":     teammate.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			teammate := specs.CreateRandomTeammate(TestAPI, team)
			TestAPI.Delete(tests.Params{
				"team":     teammate.TeamUUID,
				"teammate": teammate.UUID,
			}, nil)

			updatePayload := fixtures.RandomTeamCreatePayload()
			delete(updatePayload, "uuid")
			delete(updatePayload, "team_uuid")
			updatePayload["resource_version"] = teammate.Version
			updatePayload["role_at_team"] = teammate.RoleAtTeam
			TestAPI.Update(tests.Params{
				"team":         teammate.TeamUUID,
				"teammate":     teammate.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})

	It("Creating manager", func() {
		createPayload := fixtures.RandomTeammateCreatePayload(team)
		createPayload["role_at_team"] = model.ManagerRole

		createdData := TestAPI.Create(tests.Params{
			"team": team.UUID,
		}, nil, createPayload)

		checkTeamHasGroupsWithTeammate(cfg.FlantTenantUUID, team.UUID, createdData.Get("teammate.uuid").String(),
			true, usecase.DirectMembersGroupType, usecase.DirectManagersGroupType)
	})
})

func checkTeammateInAllFlantGroup(cfg *config.FlantFlowConfig, teammateUUID model2.UserUUID, shouldBe bool) {
	allFlantGroupResp := GroupAPI.Read(tests.Params{
		"tenant": cfg.FlantTenantUUID,
		"group":  cfg.AllFlantGroupUUID,
	}, nil)

	userFound := false
	for _, userUUID := range allFlantGroupResp.Get("group.users").Array() {
		if userUUID.String() == teammateUUID {
			userFound = true
		}
	}
	if shouldBe {
		Expect(userFound).To(BeTrue(), fmt.Sprintf("after creating teammate [%s], he should be in users of "+
			"allFlantGroup [%s], got users:\n%s", teammateUUID, cfg.AllFlantGroupUUID, allFlantGroupResp.Get("group.users").String()))
	} else {
		Expect(userFound).To(BeFalse(), fmt.Sprintf("after deleting teammate [%s], he should be deleted from users of "+
			"allFlantGroup [%s], got users:\n%s", teammateUUID, cfg.AllFlantGroupUUID, allFlantGroupResp.Get("group.users").String()))
	}

	memberFound := false
	for _, member := range allFlantGroupResp.Get("group.members").Array() {
		if member.Get("uuid").String() == teammateUUID {
			memberFound = true
		}
	}
	if shouldBe {
		Expect(memberFound).To(BeTrue(), fmt.Sprintf("after creating teammate [%s], he should be in members of "+
			"allFlantGroup [%s], got memebers:\n%s", teammateUUID, cfg.AllFlantGroupUUID, allFlantGroupResp.Get("group.members").String()))
	} else {
		Expect(userFound).To(BeFalse(), fmt.Sprintf("after deleting teammate [%s], he should be deleted from members of "+
			"allFlantGroup [%s], got users:\n%s", teammateUUID, cfg.AllFlantGroupUUID, allFlantGroupResp.Get("group.members").String()))
	}
}

func checkTeamHasGroupsWithTeammate(flantTenantUUID model2.TenantUUID, teamUUID model.TeamUUID,
	teammateUUID model2.UserUUID, expectHas bool, groupTypes ...string) {
	respData := TeamAPI.Read(tests.Params{
		"team": teamUUID,
	}, nil)
	Expect(respData.Map()).To(HaveKey("team"))
	teamData := respData.Get("team")
	Expect(teamData.Map()).To(HaveKey("groups"))
	Expect(teamData.Get("groups").Array()).To(HaveLen(3)) // should be updated after adding new group_types
	groups := teamData.Get("groups").Array()
	for _, checkingGroupType := range groupTypes {
		for _, g := range groups {
			Expect(g.Map()).To(HaveKey("type"))
			if g.Get("type").String() == checkingGroupType {
				Expect(g.Map()).To(HaveKey("uuid"))
				groupUUID := g.Get("uuid").String()
				specs.CheckGroupHasUser(GroupAPI, flantTenantUUID, groupUUID, teammateUUID, expectHas)
			}
		}
	}
}

func tryCreateRandomTeammateAtTeamWithIdentifier(team model.Team,
	teammateIdentifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomTeammateCreatePayload(team)
	payload["identifier"] = teammateIdentifier

	params := tests.Params{
		"team":         team.UUID,
		"expectStatus": tests.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}
