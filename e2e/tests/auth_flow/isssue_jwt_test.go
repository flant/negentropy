package auth_flow

import (
	"time"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var _ = Describe("Token issuing", func() {
	Context("Multipass", func() {
		var user *iam.User
		var multipassJWT string
		var multipass *iam.Multipass
		var client *api.Client

		Context("not issue", func() {
			It("if multipass is not exists", func() {
				prolongUserMultipass(false, uuid.New(), iamAuthClientWithRoot)
			})

			Context("jwt disabling", func() {
				BeforeEach(func() {
					_, multipass, _ = PrepareUserAndMultipass(true)

					switchJwt(false)
				})
				AfterEach(func() {
					switchJwt(true)
				})

				It("if jwt is disabled", func() {
					prolongUserMultipass(false, multipass.UUID, iamAuthClientWithRoot)
				})
			})

			It("if user don't own the multipass", func() {
				_, _, multipassJWT = PrepareUserAndMultipass(true)
				client = clientByMultipass(multipassJWT)
				_, strangersMultipass, _ := PrepareUserAndMultipass(true)

				prolongUserMultipass(false, strangersMultipass.UUID, client)
			})

			It("if user has been deleted", func() {
				user, multipass, multipassJWT = PrepareUserAndMultipass(true)
				client = clientByMultipass(multipassJWT)
				deleteUser(user)

				prolongUserMultipass(false, multipass.UUID, client)
			})
		})

		Context("succesfull issue", func() {
			parseToken := func(token string) (*jwt.JSONWebToken, jwt.Claims) {
				// correct parse jwt
				parsed, err := jwt.ParseSigned(token)
				Expect(err).ToNot(HaveOccurred())

				// validate expiration and issuer
				claims := jwt.Claims{}
				err = parsed.UnsafeClaimsWithoutVerification(&claims)
				Expect(err).ToNot(HaveOccurred())

				return parsed, claims
			}

			JustBeforeEach(func() {
				_, multipass, multipassJWT = PrepareUserAndMultipass(true)
				client = clientByMultipass(multipassJWT)
			})

			It("verifies with jwks", func() {
				token := prolongUserMultipass(true, multipass.UUID, client)

				parsed, claims := parseToken(token)

				jwks := getJwks()

				var err error
				for _, k := range jwks.Keys {
					// correct signs
					data := map[string]interface{}{}
					err = parsed.Claims(k, &data)
					if err == nil {
						break
					}
				}

				Expect(err).ToNot(HaveOccurred())

				err = claims.Validate(jwt.Expected{
					Issuer: "https://auth.negentropy.flant.com/",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			It("generates new JTI", func() {
				token := prolongUserMultipass(true, multipass.UUID, client)

				_, claimsNew := parseToken(token)
				_, claimsOld := parseToken(multipassJWT)

				Expect(claimsNew.ID).ToNot(BeEquivalentTo(claimsOld.ID))
			})
		})
	})
})

func clientByMultipass(multipassJWT string) *api.Client {
	auth := login(true, map[string]interface{}{
		"method": "multipass",
		"jwt":    multipassJWT,
	})

	token := auth.ClientToken
	Expect(token).ToNot(BeEmpty())

	return configure.GetClientWithToken(token, authVaultAddr)
}

func deleteUser(user *iam.User) {
	_, err := iamClientWithRoot.Logical().Delete(lib.IamPluginPath + "/tenant/" + user.TenantUUID + "/user/" + user.UUID)
	Expect(err).ToNot(HaveOccurred())

	time.Sleep(2 * time.Second)
}

func deleteServiceAccount(sa *iam.ServiceAccount) {
	_, err := iamClientWithRoot.Logical().Delete(lib.IamPluginPath + "/tenant/" + sa.TenantUUID + "/service_account/" + sa.UUID)
	Expect(err).ToNot(HaveOccurred())

	time.Sleep(2 * time.Second)
}
