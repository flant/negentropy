package jwtauth

import (
	"fmt"
	"io"
	"sort"
	"testing"

	"github.com/hashicorp/vault/api"
	"gotest.tools/assert"
)

const testJwtTypeOptionSchemaValid = `type: object
additionalProperties: false
required: [apiVersion, kind]
properties:
  apiVersion:
    type: string
    enum: [negentropy.io/v1, negentropy.io/v1alpha1]
  kind:
    type: string
    minLength: 2
  type:
    type: object
    required: [provider]
    additionalProperties: false
    properties:
      provider:
        type: string
        enum:
        - "A"
        - "B"
  CIDR:
    type: string
`

func jwtTypePathRequester(t *testing.T) apiRequester {
	return newVaultRequester(t, HttpPathJwtType)
}

func createJWTType(t *testing.T, params map[string]interface{}) (string, *api.Response) {
	name := randomStr()
	resp := jwtTypePathRequester(t).Create(name, params)
	return name, resp
}

func mustCreateUpdateJWTType(t *testing.T, body map[string]interface{}) string {
	name, resp := createJWTType(t, body)
	code := resp.StatusCode
	if code != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("incorrect response code after creating: %v: %v", code, string(b))
	}

	return name
}

func getJWTType(t *testing.T, name string) *api.Response {
	return jwtTypePathRequester(t).Get(name)
}

func cleanAllJWTTypes(t *testing.T) {
	keys, _ := jwtTypePathRequester(t).ListKeys()

	for _, n := range keys {
		jwtTypePathRequester(t).Delete(n)
	}
}

func assertJwtType(t *testing.T, resp *api.Response, data map[string]interface{}) {
	respData := extractResponseData(t, resp)

	if uuid, ok := respData["uuid"]; !ok || uuid == "" {
		t.Errorf("incorrect uuid")
	}

	for k, v := range data {
		rv, ok := respData[k]

		if !ok {
			t.Errorf("has not key '%s' in response", k)
		}

		assert.DeepEqual(t, rv, v)
	}
}

func TestJWTTypePath(t *testing.T) {
	skipNoneDev(t)

	jWTTypeCreate(t)
	jWTTypeGet(t)
	jWTTypeDelete(t)
	jWTTypeUpdate(t)
	jWTTypeList(t)
}

func jWTTypeCreate(t *testing.T) {
	t.Run("creating successful", func(t *testing.T) {
		cases := []struct {
			title string
			body  map[string]interface{}
		}{
			{
				title: "creates with all supported fields",
				body: map[string]interface{}{
					"ttl":            "1s",
					"options_schema": testJwtTypeOptionSchemaValid,
				},
			},

			{
				title: "creates without 'options_schema'",
				body: map[string]interface{}{
					"ttl": "1s",
				},
			},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				name := mustCreateUpdateJWTType(t, c.body)

				resp := getJWTType(t, name)
				assertResponseCode(t, resp, 200)

				assertJwtType(t, resp, c.body)
			})
		}
	})

	t.Run("creating failed", func(t *testing.T) {
		cases := []struct {
			title string
			body  map[string]interface{}
		}{
			{
				title: "without ttl",
				body: map[string]interface{}{
					"options_schema": testJwtTypeOptionSchemaValid,
				},
			},

			{
				title: "with ttl less than 1 second",
				body: map[string]interface{}{
					"ttl": "0s",
				},
			},

			{
				title: "if options_schema not is openapi3",
				body: map[string]interface{}{
					"ttl":            "1s",
					"options_schema": "invalid",
				},
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprintf("returns 400 %s", c.title), func(t *testing.T) {
				_, resp := createJWTType(t, c.body)
				assertResponseCode(t, resp, 400)
			})

			t.Run(fmt.Sprintf("does not creating %s", c.title), func(t *testing.T) {
				name, _ := createJWTType(t, c.body)
				resp := getJWTType(t, name)
				assertResponseCode(t, resp, 404)
			})
		}
	})
}

func jWTTypeGet(t *testing.T) {
	t.Run("successful getting", func(t *testing.T) {
		body := map[string]interface{}{
			"ttl":            "1s",
			"options_schema": testJwtTypeOptionSchemaValid,
		}
		name := mustCreateUpdateJWTType(t, body)

		t.Run("returns 200 if exists", func(t *testing.T) {
			resp := getJWTType(t, name)
			assertResponseCode(t, resp, 200)
		})

		t.Run("gets all supported fields", func(t *testing.T) {
			resp := getJWTType(t, name)
			assertJwtType(t, resp, body)
		})
	})

	t.Run("returns 404 if does not exists", func(t *testing.T) {
		const name = "not_exists"
		resp := getJWTType(t, name)
		assertResponseCode(t, resp, 404)
	})
}

func jWTTypeList(t *testing.T) {
	cleanAllJWTTypes(t)

	names := make([]string, 0)
	for i := 1; i <= 2; i++ {
		body := map[string]interface{}{
			"ttl":            "1s",
			"options_schema": testJwtTypeOptionSchemaValid,
		}
		name := mustCreateUpdateJWTType(t, body)
		names = append(names, name)
	}

	sort.Strings(names)

	t.Run("returns list of names of exists jwt types", func(t *testing.T) {
		keys, resp := jwtTypePathRequester(t).ListKeys()

		assertResponseCode(t, resp, 200)

		sort.Strings(keys)
		assert.DeepEqual(t, names, keys)
	})
}

func jWTTypeUpdate(t *testing.T) {
	t.Run("successful updating", func(t *testing.T) {
		cases := []struct {
			title string
			body  map[string]interface{}
		}{
			{
				title: "all in",
				body: map[string]interface{}{
					"ttl":            "1s",
					"options_schema": testJwtTypeOptionSchemaValid,
				},
			},

			{
				title: "only ttl",
				body: map[string]interface{}{
					"ttl": "1s",
				},
			},

			{
				title: "only options_schema",
				body: map[string]interface{}{
					"options_schema": testJwtTypeOptionSchemaValid,
				},
			},
		}

		for _, c := range cases {
			originalBody := map[string]interface{}{
				"ttl":            "1s",
				"options_schema": testJwtTypeOptionSchemaValid,
			}

			name := mustCreateUpdateJWTType(t, originalBody)
			t.Run(fmt.Sprintf("updates %s", c.title), func(t *testing.T) {
				resp := jwtTypePathRequester(t).Update(name, c.body)
				assertResponseCode(t, resp, 200)

				resp = getJWTType(t, name)
				assertResponseCode(t, resp, 200)

				assertJwtType(t, resp, c.body)
			})
		}
	})

	t.Run("updating failed", func(t *testing.T) {
		originalBody := map[string]interface{}{
			"ttl":            "1s",
			"options_schema": testJwtTypeOptionSchemaValid,
		}

		cases := []struct {
			title string
			body  map[string]interface{}
		}{
			{
				title: "with ttl less than 1 second",
				body: map[string]interface{}{
					"ttl": "0s",
				},
			},

			{
				title: "if options_schema not is openapi3",
				body: map[string]interface{}{
					"options_schema": "invalid",
				},
			},
		}

		for _, c := range cases {
			name := mustCreateUpdateJWTType(t, originalBody)

			t.Run(fmt.Sprintf("does not update %s", c.title), func(t *testing.T) {
				resp := jwtTypePathRequester(t).Update(name, c.body)
				assertResponseCode(t, resp, 400)
			})
		}
	})
}

func jWTTypeDelete(t *testing.T) {
	t.Run("successful deleting", func(t *testing.T) {
		originalBody := map[string]interface{}{
			"ttl":            "1s",
			"options_schema": testJwtTypeOptionSchemaValid,
		}
		name := mustCreateUpdateJWTType(t, originalBody)

		t.Run("returns 204 if delete exists jwt type", func(t *testing.T) {
			resp := jwtTypePathRequester(t).Delete(name)
			assertResponseCode(t, resp, 204)
		})

		t.Run("does not found jwt type after delete", func(t *testing.T) {
			resp := getJWTType(t, name)
			assertResponseCode(t, resp, 404)
		})
	})

	t.Run("returns 204 if try to delete none exists jwt type", func(t *testing.T) {
		resp := jwtTypePathRequester(t).Delete("not_exists")
		assertResponseCode(t, resp, 204)
	})
}
