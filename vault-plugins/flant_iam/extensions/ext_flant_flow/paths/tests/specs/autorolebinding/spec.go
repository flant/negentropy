package autorb

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

var (
	TeamAPI tests.TestAPI
	RoleAPI tests.TestAPI

	TenantAPI tests.TestAPI
	ConfigAPI testapi.ConfigAPI

	GroupAPI tests.TestAPI

	RoleBindingAPI tests.TestAPI
)

var _ = Describe("Teammate", func() {
	var cfg *config.FlantFlowConfig

	BeforeSuite(func() {
		cfg = specs.ConfigureFlantFlow(TenantAPI, RoleAPI, TeamAPI, GroupAPI, ConfigAPI)
	}, 1.0)

	It("if flant-all roles are set, some roles are set for flant-all group", func() {
		Expect(cfg.AllFlantGroupRoleBindingUUID).ToNot(Equal(""))
		RoleBindingAPI.Read(tests.Params{
			"tenant":       cfg.FlantTenantUUID,
			"role_binding": cfg.AllFlantGroupRoleBindingUUID,
			"expectPayload": func(json gjson.Result) {
				fmt.Printf(json.String())
				rb := json.Get("role_binding")
				members := rb.Get("members").Array()
				Expect(members).To(HaveLen(1))
				Expect(members[0].Get("uuid").String()).To(Equal(cfg.AllFlantGroupUUID))
				roles := rb.Get("roles").Array()
				Expect(roles).To(HaveLen(1))
				Expect(rb.Get("description").String()).To(Equal("all-flant group rolebinding"))
			},
		}, nil)
	})
})
