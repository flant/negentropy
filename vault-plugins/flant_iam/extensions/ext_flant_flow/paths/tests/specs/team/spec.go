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
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI tests.TestAPI

	RoleAPI   tests.TestAPI
	ConfigAPI testapi.ConfigAPI

	GroupAPI    tests.TestAPI
	TeammateAPI tests.TestAPI
)

var _ = Describe("Team", func() {
	var flantFlowCfg *config.FlantFlowConfig
	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(RoleAPI, TestAPI, ConfigAPI)
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
		var directManagersGroupUUID model.GroupUUID
		var managersGroupUUID model.GroupUUID

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
				Expect(teamData.Get("groups").Array()).To(HaveLen(3))
				directLinkedGroup := teamData.Get("groups").Array()[0]
				Expect(directLinkedGroup.Map()).To(HaveKey("type"))
				Expect(directLinkedGroup.Get("type").String()).To(Equal(usecase.DirectMembersGroupType))
				Expect(directLinkedGroup.Map()).To(HaveKey("uuid"))
				directGroupUUID = directLinkedGroup.Get("uuid").String()

				directManagersLinkedGroup := teamData.Get("groups").Array()[1]
				Expect(directManagersLinkedGroup.Map()).To(HaveKey("type"))
				Expect(directManagersLinkedGroup.Get("type").String()).To(Equal(usecase.DirectManagersGroupType))
				Expect(directManagersLinkedGroup.Map()).To(HaveKey("uuid"))
				directManagersGroupUUID = directManagersLinkedGroup.Get("uuid").String()

				managersLinkedGroup := teamData.Get("groups").Array()[2]
				Expect(managersLinkedGroup.Map()).To(HaveKey("type"))
				Expect(managersLinkedGroup.Get("type").String()).To(Equal(usecase.ManagersGroupType))
				Expect(managersLinkedGroup.Map()).To(HaveKey("uuid"))
				managersGroupUUID = managersLinkedGroup.Get("uuid").String()
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)
		checkAtTenantExistsGroups(flantFlowCfg.FlantTenantUUID, directGroupUUID, directManagersGroupUUID, managersGroupUUID)
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

	Context("Deletion rules", func() {
		Context("Teammate on board aspect", func() {
			It("Team can't be deleted, if has teammate on board", func() {
				team := specs.CreateRandomTeam(TestAPI)
				specs.CreateRandomTeammate(TeammateAPI, team)

				TestAPI.Delete(tests.Params{
					"team":         team.UUID,
					"expectStatus": tests.ExpectExactStatus(400),
				}, nil)
			})
			It("Team can be deleted, if has teammate is deleted", func() {
				team := specs.CreateRandomTeam(TestAPI)
				teammate := specs.CreateRandomTeammate(TeammateAPI, team)
				TeammateAPI.Delete(tests.Params{
					"team":     teammate.TeamUUID,
					"teammate": teammate.UUID,
				}, nil)

				TestAPI.Delete(tests.Params{
					"team": team.UUID,
				}, nil)
			})
		})
		Context("Parent team link aspect", func() {
			It("Team can be deleted, if has parent", func() {
				parentTeam := specs.CreateRandomTeam(TestAPI)
				childTeam := specs.CreateRandomTeamWithParent(TestAPI, parentTeam.UUID)

				TestAPI.Delete(tests.Params{
					"team": childTeam.UUID,
				}, nil)

				TestAPI.Read(tests.Params{
					"team": parentTeam.UUID,
					"expectPayload": func(json gjson.Result) {
						Expect(json.Get("team.archiving_timestamp").Int()).To(Equal(int64(0)))
					},
				}, nil)
			})
			It("Team can not be deleted, if has child team", func() {
				parentTeam := specs.CreateRandomTeam(TestAPI)
				specs.CreateRandomTeamWithParent(TestAPI, parentTeam.UUID)

				TestAPI.Delete(tests.Params{
					"team":         parentTeam.UUID,
					"expectStatus": tests.ExpectExactStatus(400),
				}, nil)
			})
		})
	})

	Context("Rules of creation groups nested teams", func() {
		It("After creation child team^ it's managers group contains own direct_managers_group and parent deirect_managers_group", func() {
			parentTeam := specs.CreateRandomTeam(TestAPI)
			childTeam := specs.CreateRandomTeamWithParent(TestAPI, parentTeam.UUID)
			managersChildGroupUUID := groupsMap(childTeam.Groups)[usecase.ManagersGroupType]
			directManagersChildGroupUUID := groupsMap(childTeam.Groups)[usecase.DirectManagersGroupType]
			directManagersParentGroupUUID := groupsMap(parentTeam.Groups)[usecase.DirectManagersGroupType]

			GroupAPI.Read(tests.Params{
				"tenant": flantFlowCfg.FlantTenantUUID,
				"group":  managersChildGroupUUID,
				"expectPayload": func(json gjson.Result) {
					groupData := json.Get("group")
					Expect(groupData.Map()).To(HaveKey("groups"))
					Expect(groupData.Get("groups").Array()).To(HaveLen(2))
					groups := groupData.Get("groups").Array()
					Expect(groups[0].String()).To(Equal(directManagersParentGroupUUID))
					Expect(groups[1].String()).To(Equal(directManagersChildGroupUUID))
				},
			}, nil)
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

func checkAtTenantExistsGroups(tenantUUID string, groupUUIDs ...string) {
	for _, g := range groupUUIDs {
		GroupAPI.Read(tests.Params{
			"tenant": tenantUUID,
			"group":  g,
			"expectPayload": func(json gjson.Result) {
				groupData := json.Get("group")
				Expect(groupData.Map()).To(HaveKey("uuid"))
			},
		}, nil)
	}
}

// groupsMap returns map[GroupType]GroupUUID
func groupsMap(groups []model2.LinkedGroup) map[string]model.GroupUUID {
	result := map[string]model.GroupUUID{}
	for _, g := range groups {
		result[g.Type] = g.GroupUUID
	}
	return result
}
