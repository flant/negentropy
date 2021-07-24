package flow

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"gopkg.in/square/go-jose.v2/jwt"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

var _ = Describe("Token issuing", func() {
	Context("Multipass", func() {
		var multipassJwt string
		var user *iam.User
		var multipass *iam.Multipass

		Context("not issue", func() {
			It("if multipass is not exists", func() {
				prolongUserMultipass(false, uuid.New(), iamAuthClient)
			})

			Context("jwt disabling", func() {
				BeforeEach(func() {
					user = createUser()
					multipass, _ = createUserMultipass(user)

					switchJwt(false)
				})
				AfterEach(func() {
					switchJwt(true)
				})

				It("if jwt is disabled", func() {
					prolongUserMultipass(false, multipass.UUID, iamAuthClient)
				})
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
				user = createUser()
				multipass, multipassJwt = createUserMultipass(user)
			})

			It("verifies with jwks", func() {
				token := prolongUserMultipass(true, multipass.UUID, iamAuthClient)

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
				token := prolongUserMultipass(true, multipass.UUID, iamAuthClient)

				_, claimsNew := parseToken(token)
				_, claimsOld := parseToken(multipassJwt)

				Expect(claimsNew.ID).ToNot(BeEquivalentTo(claimsOld.ID))
			})
		})
	})
})
