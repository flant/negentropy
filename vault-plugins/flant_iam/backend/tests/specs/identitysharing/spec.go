package identitysharing

import (
	"encoding/json"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TenantAPI api.TestAPI
	GroupAPI  api.TestAPI
	TestAPI   api.TestAPI
)

var _ = Describe("Identity sharing", func() {
	var (
		sourceTenantID, targetTenantID string
		group                          model.Group
	)

	BeforeEach(func() {
		t1 := specs.CreateRandomTenant(TenantAPI)
		sourceTenantID = t1.UUID
		t2 := specs.CreateRandomTenant(TenantAPI)
		targetTenantID = t2.UUID
		group = specs.CreateRandomEmptyGroup(GroupAPI, sourceTenantID)
	})

	var createdData gjson.Result

	It("can be created", func() {
		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				is := json.Get("identity_sharing")

				Expect(is.Map()).To(HaveKey("uuid"))
				Expect(is.Map()).To(HaveKey("source_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("destination_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("destination_tenant_identifier"))
				Expect(is.Map()).To(HaveKey("groups"))
				Expect(is.Map()).To(HaveKey("origin"))
				Expect(is.Get("groups").Array()).To(HaveLen(1))
			},
			"tenant": sourceTenantID,
		}
		data := map[string]interface{}{
			"destination_tenant_uuid": targetTenantID,
			"groups":                  []string{group.UUID},
		}
		createdData = TestAPI.Create(params, url.Values{}, data)
	})

	It("can be read", func() {
		TestAPI.Read(api.Params{
			"uuid":   createdData.Get("identity_sharing.uuid").String(),
			"tenant": sourceTenantID,
			"expectPayload": func(json gjson.Result) {
				Expect(createdData).To(Equal(json))
			},
		}, nil)
	})

	It("can be listed", func() {
		createdIS := createIdentitySharing(TestAPI, sourceTenantID, targetTenantID, group.UUID)
		list := TestAPI.List(api.Params{
			"tenant": sourceTenantID,
		}, url.Values{})
		Expect(list.Get("identity_sharings").Array()).To(HaveLen(1))
		Expect(list.Get("identity_sharings").Array()[0].Get("uuid").String()).To(BeEquivalentTo(createdIS.UUID))
	})

	It("can be updated", func() {
		createdIS := createIdentitySharing(TestAPI, sourceTenantID, targetTenantID, group.UUID)
		group2 := specs.CreateRandomEmptyGroup(GroupAPI, sourceTenantID)
		updatePayload := map[string]interface{}{}
		updatePayload["destination_tenant_uuid"] = targetTenantID
		updatePayload["resource_version"] = createdIS.Version
		updatePayload["groups"] = []string{group.UUID, group2.UUID}

		updateData := TestAPI.Update(api.Params{
			"tenant": group.TenantUUID,
			"uuid":   createdIS.UUID,
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant": group.TenantUUID,
			"uuid":   createdIS.UUID,
			"expectPayload": func(json gjson.Result) {
				isData := json.Get("identity_sharing")
				specs.IsSubsetExceptKeys(updateData.Get("identity_sharing"), isData)
				Expect(isData.Map()).To(HaveKey("origin"))
				Expect(isData.Get("origin").String()).To(Equal("iam"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		createdIS := createIdentitySharing(TestAPI, sourceTenantID, targetTenantID, group.UUID)
		TestAPI.Delete(api.Params{
			"uuid":   createdIS.UUID,
			"tenant": sourceTenantID,
		}, nil)

		deletedISData := TestAPI.Read(api.Params{
			"uuid":         createdIS.UUID,
			"tenant":       sourceTenantID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedISData.Get("identity_sharing.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be created with privileged", func() {
		t1 := specs.CreateRandomTenant(TenantAPI)
		sourceTenantID = t1.UUID
		t2 := specs.CreateRandomTenant(TenantAPI)
		targetTenantID = t2.UUID
		group = specs.CreateRandomEmptyGroup(GroupAPI, sourceTenantID)

		originalUUID := uuid.New()

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				is := json.Get("identity_sharing")

				Expect(is.Map()).To(HaveKey("uuid"))
				Expect(is.Map()["uuid"].String()).To(Equal(originalUUID))
				Expect(is.Map()).To(HaveKey("source_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("destination_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("groups"))
				Expect(is.Get("groups").Array()).To(HaveLen(1))
			},
			"tenant": sourceTenantID,
		}
		data := map[string]interface{}{
			"destination_tenant_uuid": targetTenantID,
			"groups":                  []string{group.UUID},
			"uuid":                    originalUUID,
		}
		createdData = TestAPI.CreatePrivileged(params, url.Values{}, data)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			is := createIdentitySharing(TestAPI, sourceTenantID, targetTenantID, group.UUID)
			TestAPI.Delete(api.Params{
				"uuid":   is.UUID,
				"tenant": sourceTenantID,
			}, nil)

			TestAPI.Delete(api.Params{
				"uuid":         is.UUID,
				"tenant":       sourceTenantID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			is := createIdentitySharing(TestAPI, sourceTenantID, targetTenantID, group.UUID)
			TestAPI.Delete(api.Params{
				"uuid":   is.UUID,
				"tenant": sourceTenantID,
			}, nil)

			updatePayload := map[string]interface{}{
				"destination_tenant_uuid": targetTenantID,
				"resource_version":        is.Version,
				"groups":                  is.Groups,
			}
			TestAPI.Update(api.Params{
				"tenant":       group.TenantUUID,
				"uuid":         is.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func createIdentitySharing(identitySharingAPI api.TestAPI, sourceTenantUUID model.TenantUUID,
	targetTenantID model.TenantUUID, groupsUUIDS ...model.GroupUUID) model.IdentitySharing {
	payload := map[string]interface{}{
		"destination_tenant_uuid": targetTenantID,
		"groups":                  groupsUUIDS,
	}
	createdData := identitySharingAPI.Create(api.Params{
		"tenant": sourceTenantUUID,
	}, url.Values{}, payload)
	rawIS := createdData.Get("identity_sharing")
	data := []byte(rawIS.String())
	var is usecase.DenormalizedIdentitySharing
	err := json.Unmarshal(data, &is)
	Expect(err).ToNot(HaveOccurred())
	groups := make([]model.GroupUUID, 0, len(is.Groups))
	for _, g := range is.Groups {
		groups = append(groups, g.UUID)
	}
	return model.IdentitySharing{
		ArchiveMark:           is.ArchiveMark,
		UUID:                  is.UUID,
		SourceTenantUUID:      is.SourceTenantUUID,
		DestinationTenantUUID: is.DestinationTenantUUID,
		Version:               is.Version,
		Origin:                is.Origin,
		Groups:                groups,
	}
}
