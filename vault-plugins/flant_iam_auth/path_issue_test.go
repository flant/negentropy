package jwtauth

import (
	"fmt"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func jwtIssueRequester(t *testing.T) apiRequester {
	return newVaultRequester(t, HttpPathIssue)
}

func jwtRequester(t *testing.T) apiRequester {
	return newVaultRequester(t, "jwt")
}

func enableJwt(t *testing.T) {
	resp := jwtRequester(t).Update("enable", nil)
	assertResponseCode(t, resp, 200)
}

func disableJwt(t *testing.T) {
	resp := jwtRequester(t).Update("disable", nil)
	assertResponseCode(t, resp, 200)
}

func getJWKS(t *testing.T) *jose.JSONWebKeySet {
	resp := newVaultRequester(t, "jwks").Get("")
	assertResponseCode(t, resp, 200)
	keys := jose.JSONWebKeySet{}
	extractResponseDataT(t, resp, &keys)
	return &keys
}

func issueJwt(t *testing.T, name string, params map[string]interface{}) *api.Response {
	return jwtIssueRequester(t).Update(fmt.Sprintf("jwt/%s", name), params)
}

func TestIssuePath(t *testing.T) {
	skipNoneDev(t)

	issuePathJWT(t)
}

func issuePathJWT(t *testing.T) {
	jwtTypeBody := map[string]interface{}{
		"ttl":            "200s",
		"options_schema": testJwtTypeOptionSchemaValid,
	}
	name := mustCreateUpdateJWTType(t, jwtTypeBody)

	validOpts := map[string]interface{}{
		"apiVersion": "negentropy.io/v1",
		"kind":       "kind",
		"type": map[string]interface{}{
			"provider": "A",
		},
		"CIDR": "aaaaaa",
	}

	validBody := map[string]interface{}{
		"options": validOpts,
	}

	t.Run("subscribes successful", func(t *testing.T) {
		enableJwt(t)
		jwks := getJWKS(t)

		resp := issueJwt(t, name, validBody)
		assertResponseCode(t, resp, 200)

		// has jwt key in response
		data := extractResponseData(t, resp)
		assert.Contains(t, data, "jwt")

		// correct parse jwt
		token := data["jwt"].(string)
		parsed, err := jwt.ParseSigned(token)
		assert.NoError(t, err)

		// correct signs
		data = map[string]interface{}{}
		err = parsed.Claims(jwks.Keys[0], &data)
		assert.NoError(t, err)

		// validate expiration and issuer
		claims := jwt.Claims{}
		err = parsed.UnsafeClaimsWithoutVerification(&claims)
		assert.NoError(t, err)
		err = claims.Validate(jwt.Expected{
			Issuer: "https://auth.negentropy.flant.com/",
		})
		assert.NoError(t, err)

		// validate correct options
		for k, v := range validOpts {
			assert.Contains(t, data, k)
			assert.Equal(t, v, data[k])
		}
	})

	t.Run("creating failed", func(t *testing.T) {
		t.Run("returns 400 when jwt feature is disabled", func(t *testing.T) {
			enableJwt(t)
			name := mustCreateUpdateJWTType(t, jwtTypeBody)
			disableJwt(t)
			resp := issueJwt(t, name, validBody)
			assertResponseCode(t, resp, 400)
		})

		t.Run("returns 404 for not exists jwt type", func(t *testing.T) {
			enableJwt(t)
			resp := issueJwt(t, "not_exists", validBody)
			assertResponseCode(t, resp, 404)
		})

		incorrectOptsCases := []struct {
			title string
			opts  map[string]interface{}
		}{
			{
				title: "opts with not known field",
				opts: map[string]interface{}{
					"incorrect_field": 1,
					"apiVersion":      "v1",
					"kind":            "kind",
					"type": map[string]interface{}{
						"provider": "a",
					},
					"CIDR": "aaaaaa",
				},
			},

			{
				title: "opts with not exists required fields",
				opts: map[string]interface{}{
					"apiVersion": "v1",
					"type": map[string]interface{}{
						"provider": "a",
					},
					"CIDR": "aaaaaa",
				},
			},

			{
				title: "opts with incorrect field constraint",
				opts: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "k",
					"type": map[string]interface{}{
						"provider": "a",
					},
					"CIDR": "aaaaaa",
				},
			},
		}

		for _, c := range incorrectOptsCases {
			t.Run(fmt.Sprintf("returns 400 for %s", c.title), func(t *testing.T) {
				enableJwt(t)
				name := mustCreateUpdateJWTType(t, jwtTypeBody)
				resp := issueJwt(t, name, map[string]interface{}{
					"options": c.opts,
				})
				assertResponseCode(t, resp, 400)
			})
		}
	})
}
