package flow

import (
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/auth_source"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/configure"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Login", func() {
	Context("with jwt method", func() {
		var jwtData string
		var user *iam.User

		JustBeforeEach(func() {
			user = createUser()
			jwtData = auth_source.SignJWT(user.FullIdentifier, time.Now().Add(5 * time.Second), map[string]interface{}{
				"email": user.Email,
				"uuid": user.UUID,
			})
		})

		It("successful log in with jwt method", func() {
			auth := login(map[string]interface{}{
				"method": jwtMethodName,
				"jwt":    jwtData,
			})

			Expect(auth.ClientToken).ToNot(BeEmpty())

			entityID, err := identityApi.EntityApi().GetID(user.FullIdentifier)
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.EntityID).To(BeEquivalentTo(entityID))
			Expect(auth.Policies).To(BeEquivalentTo([]string{methodReaderOnlyPolicyName}))

		})

		Context("accessible", func() {
			It("does access to allowed method", func() {
				auth := login(map[string]interface{}{
					"method": jwtMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				cl := configure.GetClient(auth.ClientToken)
				method, err := cl.Logical().Read(lib.IamAuthPluginPath+"/auth_method/"+ jwtMethodName)

				Expect(err).ToNot(HaveOccurred())
				Expect(method.Data["name"].(string)).To(BeEquivalentTo(jwtMethodName))
			})

			It("does not access to allowed method", func() {
				auth := login(map[string]interface{}{
					"method": jwtMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				cl := configure.GetClient(auth.ClientToken)
				_, err := cl.Logical().Delete(lib.IamAuthPluginPath+"/auth_method/"+ jwtMethodName)

				Expect(err).To(HaveOccurred())
			})
		})
	})


	Context("with multipass", func() {
		var jwtData string
		var user *iam.User

		JustBeforeEach(func() {
			user = createUser()
			jwtData = auth_source.SignJWT(user.FullIdentifier, time.Now().Add(5 * time.Second), map[string]interface{}{
				"email": user.Email,
				"uuid": user.UUID,
			})
		})

		It("successful log in with jwt method", func() {
			auth := login(map[string]interface{}{
				"method": jwtMethodName,
				"jwt":    jwtData,
			})

			Expect(auth.ClientToken).ToNot(BeEmpty())

			entityID, err := identityApi.EntityApi().GetID(user.FullIdentifier)
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.EntityID).To(BeEquivalentTo(entityID))
			Expect(auth.Policies).To(BeEquivalentTo([]string{methodReaderOnlyPolicyName}))

		})

		Context("accessible", func() {
			It("does access to allowed method", func() {
				auth := login(map[string]interface{}{
					"method": jwtMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				cl := configure.GetClient(auth.ClientToken)
				method, err := cl.Logical().Read(lib.IamAuthPluginPath+"/auth_method/"+ jwtMethodName)

				Expect(err).ToNot(HaveOccurred())
				Expect(method.Data["name"].(string)).To(BeEquivalentTo(jwtMethodName))
			})

			It("does not access to allowed method", func() {
				auth := login(map[string]interface{}{
					"method": jwtMethodName,
					"jwt":    jwtData,
				})
				Expect(auth.ClientToken).ToNot(BeEmpty())

				cl := configure.GetClient(auth.ClientToken)
				_, err := cl.Logical().Delete(lib.IamAuthPluginPath+"/auth_method/"+ jwtMethodName)

				Expect(err).To(HaveOccurred())
			})
		})
	})
})
