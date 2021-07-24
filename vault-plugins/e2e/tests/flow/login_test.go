package flow

import (
	"time"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/auth_source"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/configure"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func assertVaultUser(user *iam.User, auth *api.SecretAuth) {
	entityID, err := identityApi.EntityApi().GetID(user.FullIdentifier)
	Expect(err).ToNot(HaveOccurred())
	Expect(auth.EntityID).To(BeEquivalentTo(entityID))
	Expect(auth.Policies).To(BeEquivalentTo([]string{methodReaderOnlyPolicyName}))
}

func assertHasAccess(token string) {
	cl := configure.GetClient(token)
	method, err := cl.Logical().Read(lib.IamAuthPluginPath + "/auth_method/" + jwtMethodName)

	Expect(err).ToNot(HaveOccurred())
	Expect(method.Data["name"].(string)).To(BeEquivalentTo(jwtMethodName))
}

var _ = Describe("Login", func() {
	Context("with jwt method", func() {
		var jwtData string
		var user *iam.User

		JustBeforeEach(func() {
			user = createUser()
			jwtData = auth_source.SignJWT(user.FullIdentifier, time.Now().Add(5*time.Second), map[string]interface{}{
				"email": user.Email,
				"uuid":  user.UUID,
			})
		})

		It("successful log in", func() {
			auth := login(true, map[string]interface{}{
				"method": jwtMethodName,
				"jwt":    jwtData,
			})

			Expect(auth.ClientToken).ToNot(BeEmpty())

			assertVaultUser(user, auth)
		})

		Context("accessible", func() {
			It("does access to allowed method", func() {
				auth := login(true, map[string]interface{}{
					"method": jwtMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				cl := configure.GetClient(auth.ClientToken)
				method, err := cl.Logical().Read(lib.IamAuthPluginPath + "/auth_method/" + jwtMethodName)

				Expect(err).ToNot(HaveOccurred())
				Expect(method.Data["name"].(string)).To(BeEquivalentTo(jwtMethodName))
			})

			It("does not access to not allowed method", func() {
				auth := login(true, map[string]interface{}{
					"method": jwtMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				cl := configure.GetClient(auth.ClientToken)
				_, err := cl.Logical().Delete(lib.IamAuthPluginPath + "/auth_method/" + jwtMethodName)

				Expect(err).To(HaveOccurred())
			})
		})
	})
	Context("with multipass", func() {
		var jwtData string
		var user *iam.User
		var multipass *iam.Multipass

		BeforeEach(func() {
			user = createUser()
			multipass, jwtData = createUserMultipass(user)
		})

		It("successful log in", func() {
			auth := login(true, map[string]interface{}{
				"method": multipassMethodName,
				"jwt":    jwtData,
			})

			Expect(auth.ClientToken).ToNot(BeEmpty())

			assertVaultUser(user, auth)
		})

		Context("accessible", func() {
			It("does access to allowed method", func() {
				auth := login(true, map[string]interface{}{
					"method": multipassMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				assertHasAccess(auth.ClientToken)
			})

			It("does not access to not allowed method", func() {
				auth := login(true, map[string]interface{}{
					"method": multipassMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				cl := configure.GetClient(auth.ClientToken)
				_, err := cl.Logical().Delete(lib.IamAuthPluginPath + "/auth_method/" + jwtMethodName)

				Expect(err).To(HaveOccurred())
			})
		})

		Context("fail login", func() {
			It("does not log in after delete multipass", func() {
				auth := login(true, map[string]interface{}{
					"method": multipassMethodName,
					"jwt":    jwtData,
				})

				Expect(auth.ClientToken).ToNot(BeEmpty())

				deleteUserMultipass(user, multipass)

				auth = login(false, map[string]interface{}{
					"method": multipassMethodName,
					"jwt":    jwtData,
				})

				Expect(auth).To(BeNil())
			})
		})

		Context("multipass prolongation", func() {
			var prolongClient *api.Client

			BeforeEach(func() {
				auth := login(true, map[string]interface{}{
					"method": multipassMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				prolongClient = configure.GetClient(auth.ClientToken)
			})

			It("successful log in after prolong multipass", func() {
				token := prolongUserMultipass(true, multipass.UUID, prolongClient)

				auth := login(true, map[string]interface{}{
					"method": multipassMethodName,
					"jwt":    token,
				})

				Expect(auth.ClientToken).ToNot(BeEmpty())

				assertVaultUser(user, auth)
			})

			It("does not log in with old multipass", func() {
				prolongUserMultipass(true, multipass.UUID, prolongClient)

				auth := login(false, map[string]interface{}{
					"method": multipassMethodName,
					"jwt":    jwtData,
				})

				Expect(auth).To(BeNil())
			})
		})
	})
})
