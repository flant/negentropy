package role_options

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
)

// we have role pult.team.manage which needs required option "team_uuid" with valid uuid value
const roleName = "pult.team.manage"

var cfg flant_iam_preparing.CheckingEnvironment

var _ = Describe("Process creating rolebinding with options", func() {
	var flantIamSuite flant_iam_preparing.Suite
	flantIamSuite.BeforeSuite()
	cfg = flantIamSuite.PrepareForLoginTesting() // no needs waiting as works with iam

	It("can create rolebinding with valid value required property", func() {
		err := tryCreateRolebinding(true, map[string]interface{}{"team_uuid": "d8602a1c-c8cb-49f9-b1e9-e6fc764a7fcc"})
		Expect(err).ToNot(HaveOccurred())
	})
	It("can't create rolebinding with invalid value  required property", func() {
		err := tryCreateRolebinding(false, map[string]interface{}{"team_uuid": "not_valid_uuid"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot create role binding:invalid value of argument: check options for role \"pult.team.manage\": Error at "))
	})
	It("can't create rolebinding without value of required property", func() {
		err := tryCreateRolebinding(false, map[string]interface{}{"team2": "d8602a1c-c8cb-49f9-b1e9-e6fc764a7fcc"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot create role binding:invalid value of argument: check options for role \"pult.team.manage\": Error at "))
	})
})

func tryCreateRolebinding(expectPositive bool, options map[string]interface{}) error {
	var err error
	params := api.Params{
		"tenant": cfg.User.TenantUUID,
	}
	params["expectPayload"] = func(json gjson.Result) {
		errMsg := json.Get("error").String()
		if errMsg != "" {
			err = fmt.Errorf("%s", errMsg)
		}
	}
	if expectPositive {
		params["expectStatus"] = api.ExpectExactStatus(201)
	} else {
		params["expectStatus"] = api.ExpectExactStatus(400)
	}

	rb := model.RoleBinding{
		TenantUUID:  cfg.User.TenantUUID,
		Version:     "",
		Description: "rolebinding for check option validation",
		ValidTill:   10_000_000_000,
		RequireMFA:  false,
		Members:     []model.MemberNotation{{Type: "user", UUID: cfg.User.UUID}},
		AnyProject:  false,
		Roles:       []model.BoundRole{{Name: roleName, Options: options}},
	}
	bytes, _ := json.Marshal(rb)
	var createPayload map[string]interface{}
	json.Unmarshal(bytes, &createPayload) //nolint:errcheck
	delete(createPayload, "valid_till")
	createPayload["ttl"] = rb.ValidTill - time.Now().Unix()
	lib.NewRoleBindingAPI(lib.NewConfiguredIamVaultClient()).Create(params, url.Values{}, createPayload)
	return err
}
