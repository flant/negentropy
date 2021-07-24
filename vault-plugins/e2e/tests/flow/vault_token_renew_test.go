package flow

import (
	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/configure"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

var _ = Describe("Renewing token", func() {
	Context("with multipass", func() {
		var jwtData string
		var user *iam.User
		var multipass *iam.Multipass
		var prolongClient *api.Client
		var token string

		BeforeEach(func() {
			user = createUser()
			multipass, jwtData = createUserMultipass(user)
			auth := login(true, map[string]interface{}{
				"method": multipassMethodName,
				"jwt":    jwtData,
			})

			token = auth.ClientToken
			Expect(token).ToNot(BeEmpty())

			prolongClient = configure.GetClient(token)
		})

		It("successful renew not expired token", func() {
			data, err := prolongClient.Auth().Token().Renew(token, 300)
			Expect(err).ToNot(HaveOccurred())
			Expect(data.Auth.ClientToken).ToNot(BeEmpty())

			assertHasAccess(data.Auth.ClientToken)
		})

		Context("does not access", func() {
			It("after prolong multipass", func() {
				prolongUserMultipass(true, multipass.UUID, prolongClient)
				_, err := prolongClient.Auth().Token().Renew(token, 300)
				Expect(err).To(HaveOccurred())
			})

			It("after remove multipass", func() {
				deleteUserMultipass(user, multipass)

				_, err := prolongClient.Auth().Token().Renew(token, 300)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
