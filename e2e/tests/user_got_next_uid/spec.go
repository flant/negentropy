package user_got_ssh_access

import (
	_ "embed"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

var flantIamSuite flant_iam_preparing.Suite

var cfg flant_iam_preparing.CheckingEnvironment

var _ = BeforeSuite(func() {
	flantIamSuite.BeforeSuite()
	Describe("configuring system", func() {
		cfg = flantIamSuite.PrepareForSSHTesting()
	})
}, 1.0)

var _ = Describe("Process of getting new user/service_account next linux uid", func() {
	var userUID int64

	It("created user (with ssh rolebinding) has some uid", func() {

		userUID = extractUID(cfg.User)
		fmt.Println(userUID)
	})

	var newUserUID int64
	It("created new user (with ssh rolebinding) has uid", func() {
		newUser := specs.CreateRandomUser(lib.NewUserAPI(flantIamSuite.IamVaultClient), cfg.Tenant.UUID)
		newGroup := specs.CreateRandomGroupWithUser(lib.NewGroupAPI(flantIamSuite.IamVaultClient), newUser.TenantUUID, newUser.UUID)
		// create rolebinding for a user in project with the ssh role
		specs.CreateRoleBinding(lib.NewRoleBindingAPI(flantIamSuite.IamVaultClient),
			iam.RoleBinding{
				TenantUUID:  newUser.TenantUUID,
				Version:     "",
				Description: "user got next uid testing",
				ValidTill:   10_000_000_000,
				RequireMFA:  false,
				Members:     newGroup.Members,
				AnyProject:  true,
				Roles:       []iam.BoundRole{{Name: "ssh.open", Options: map[string]interface{}{}}},
			})

		newUserUID = extractUID(newUser)
		fmt.Println(newUserUID)
	})

	It("new created user's UID is greater than first user's UID ", func() {
		Expect(newUserUID).To(BeNumerically(">", userUID))
	})
})

func extractUID(user iam.User) int64 {
	Expect(user.Extensions).To(HaveKey(consts.ObjectOrigin("server_access")))
	extension := user.Extensions["server_access"]
	Expect(extension.Attributes).To(HaveKey("UID"))
	uid := int64(extension.Attributes["UID"].(float64))
	return uid
}
