package jwtauth

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/vault/api"
	"gotest.tools/assert"
	"io"
	"math/rand"
	"net/url"
	"os"
	"sort"
	"testing"
	"time"
)

const testJwtTypeOptionSchemaValid = `type: object
additionalProperties: false
required: [apiVersion, kind, type]
properties:
  apiVersion:
    type: string
    enum: [negentropy.io/v1, negentropy.io/v1alpha1]
  kind:
    type: string
    enum: [ClusterConfiguration]
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
oneOf:
- properties:
    type:
       enum: ["A"]
- properties:
    type:
       enum: ["B"]
  CIDR: "no"
  required: ["CIDR"]
`

func convertResponseToListKeys(t *testing.T, resp *api.Response) []string{
	rawResp := map[string]interface{}{}
	err := resp.DecodeJSON(&rawResp)
	if err != nil {
		t.Fatalf("can not decode response %v", err)
	}

	keysIntr := rawResp["data"].(map[string]interface{})["keys"].([]interface{})

	keys := make([]string, 0)
	for _, s := range keysIntr {
		keys = append(keys, s.(string))
	}

	return keys
}

func getJWTTypePathApi() (*api.Client, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		token = "root"
	}

	client.SetToken(token)

	return client, nil
}

func requestJwtTypeName(t *testing.T, cl *api.Client, method, name string, params map[string]interface{}, q *url.Values) *api.Request {
	path := fmt.Sprintf("/v1/auth/flant_iam_auth/%s/%s", HttpPathJwtType, name)
	r := cl.NewRequest(method, path)
	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("cannot marshal request params to json: %v", err)
			return nil
		}

		reader := bytes.NewReader(raw)
		if q != nil {
			r.Params = *q
		}

		r.Body = reader
	}
	return r
}

func createJWTType(t *testing.T, params map[string]interface{}) (string, *api.Response) {
	name := randomStr()
	resp := createUpdateJWTType(t, name, params)
	return name, resp
}

func createUpdateJWTType(t *testing.T, name string, params map[string]interface{}) *api.Response {
	cl, err := getJWTTypePathApi()

	if err != nil {
		t.Fatalf("can not get client %s", err)
	}

	r := requestJwtTypeName(t, cl, "POST", name, params, nil)
	resp, err := cl.RawRequest(r)
	if resp == nil {
		t.Fatalf("error wile send request %v", err)
	}
	return resp
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
	cl, err := getJWTTypePathApi()
	if err != nil {
		t.Fatalf("can not get client %s", err)
	}

	r := requestJwtTypeName(t, cl, "GET", name, nil, nil)
	resp, err := cl.RawRequest(r)
	if resp == nil {
		t.Fatalf("can not send request %v", err)
	}

	return resp
}

func getListJWTTypes(t *testing.T) *api.Response {
	cl, err := getJWTTypePathApi()
	if err != nil {
		t.Fatalf("can not get client %s", err)
	}

	r := cl.NewRequest("GET", fmt.Sprintf("/v1/auth/flant_iam_auth/%s/", HttpPathJwtType))
	r.Params = url.Values{
		"list": []string{"true"},
	}
	resp, err := cl.RawRequest(r)
	if resp == nil {
		t.Fatalf("can not send request %v", err)
	}

	return resp
}

func deleteJWTType(t *testing.T, name string) *api.Response {
	cl, err := getJWTTypePathApi()
	if err != nil {
		t.Fatalf("can not get client %s", err)
	}

	r := requestJwtTypeName(t, cl, "DELETE", name, nil, nil)

	resp, err := cl.RawRequest(r)
	if resp == nil {
		t.Fatalf("can not send request %v", err)
	}

	return resp
}

func cleanAllJWTTypes(t *testing.T) {
	resp := getListJWTTypes(t)
	if resp.StatusCode == 404 {
		return
	}

	if resp.StatusCode != 200 {
		t.Fatalf("cannot getting all jwt types")
	}

	keys := convertResponseToListKeys(t, resp)

	for _, n := range keys {
		deleteJWTType(t, n)
	}
}

func assertJwtType(t *testing.T, resp *api.Response, data map[string]interface{}) {
	respRaw := map[string]interface{}{}
	err := resp.DecodeJSON(&respRaw)
	if err != nil {
		t.Errorf("Do not unmarshal body: %v", err)
	}

	respData := respRaw["data"].(map[string]interface{})

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

func skipNoneDev(t *testing.T) {
	if os.Getenv("VAULT_ADDR") == "" {
		t.Skip("vault does not start")
	}
}

func randomStr() string {
	rand.Seed(time.Now().UnixNano())

	entityName := make([]byte, 20)
	_, err := rand.Read(entityName)
	if err != nil {
		panic("not generate entity name")
	}

	return hex.EncodeToString(entityName)
}

func TestJWTTypePath(t *testing.T) {
	skipNoneDev(t)

	TestJWTType_Create(t)
	TestJWTType_Get(t)
	TestJWTType_Delete(t)
	TestJWTType_Update(t)
	TestJWTType_List(t)
}

func TestJWTType_Create(t *testing.T) {
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
				code := resp.StatusCode
				if code != 200 {
					t.Errorf("jwt type %v does not exists, return code: %v", name, code)
				}

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
				code := resp.StatusCode
				if code != 400 {
					t.Errorf("incorrect response code %v", code)
				}
			})

			t.Run(fmt.Sprintf("does not creating %s", c.title), func(t *testing.T) {
				name, _ := createJWTType(t, c.body)
				resp := getJWTType(t, name)
				code := resp.StatusCode
				if code != 404 {
					t.Errorf("jwt type %s must be not found wit code 404 got %v", name, code)
				}
			})
		}
	})
}

func TestJWTType_Get(t *testing.T) {
	t.Run("successful getting", func(t *testing.T) {
		body := map[string]interface{}{
			"ttl":            "1s",
			"options_schema": testJwtTypeOptionSchemaValid,
		}
		name := mustCreateUpdateJWTType(t, body)

		t.Run("returns 200 if exists", func(t *testing.T) {
			resp := getJWTType(t, name)
			code := resp.StatusCode
			if code != 200 {
				t.Errorf("jwt type %v does not exists, return code: %v", name, code)
			}
		})

		t.Run("gets all supported fields", func(t *testing.T) {
			resp := getJWTType(t, name)
			assertJwtType(t, resp, body)
		})
	})

	t.Run("returns 404 if does not exists", func(t *testing.T) {
		const name = "not_exists"
		resp := getJWTType(t, name)
		code := resp.StatusCode
		if code != 404 {
			t.Errorf("jwt type '%s' must be not exists and returns 404, return code: %v", name, code)
		}
	})

}

func TestJWTType_List(t *testing.T) {
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
		resp := getListJWTTypes(t)

		if resp.StatusCode != 200 {
			t.Errorf("cannot getting all jwt types: response code: %v", resp.StatusCode)
		}

		keys := convertResponseToListKeys(t, resp)
		sort.Strings(keys)

		assert.DeepEqual(t, names, keys)
	})
}

func TestJWTType_Update(t *testing.T) {
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
					"ttl":            "1s",
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
				resp := createUpdateJWTType(t, name, c.body)
				code := resp.StatusCode
				if code != 200 {
					t.Errorf("Incorrect response code, got %v", code)
				}

				resp = getJWTType(t, name)
				code = resp.StatusCode
				if code != 200 {
					t.Errorf("Incorrect response code, got %v", code)
				}

				assertJwtType(t, resp, c.body)
			})
		}

	})

	t.Run("updating failed", func(t *testing.T) {
		t.Run("returns 404 if not exists", func(t *testing.T) {

		})

		cases := []struct {
			title string
			body  map[string]interface{}
		}{
			{
				title: "with ttl less than 1 second",
				body: map[string]interface{}{
					"ttl":            "0s",
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
			originalBody := map[string]interface{}{
				"ttl":            "1s",
				"options_schema": testJwtTypeOptionSchemaValid,
			}

			name := mustCreateUpdateJWTType(t, originalBody)

			t.Run(fmt.Sprintf("does not update %s", c.title), func(t *testing.T) {
				resp := createUpdateJWTType(t, name, c.body)
				code := resp.StatusCode
				if code != 400 {
					t.Errorf("Incorrect response code, got %v", code)
				}
			})
		}
	})
}

func TestJWTType_Delete(t *testing.T) {
	t.Run("successful deleting", func(t *testing.T) {
		originalBody := map[string]interface{}{
			"ttl":            "1s",
			"options_schema": testJwtTypeOptionSchemaValid,
		}
		name := mustCreateUpdateJWTType(t, originalBody)

		t.Run("returns 204 if delete exists jwt type", func(t *testing.T) {
			resp := deleteJWTType(t, name)
			code := resp.StatusCode
			if code != 204 {
				t.Errorf("Incorrect response code, got %v", code)
			}
		})

		t.Run("does not found jwt type after delete", func(t *testing.T) {
			resp := getJWTType(t, name)
			code := resp.StatusCode
			if code != 404 {
				t.Errorf("Incorrect response code, got %v", code)
			}
		})
	})

	t.Run("returns 204 if try to delete none exists jwt type", func(t *testing.T) {
		resp := deleteJWTType(t, "not_exists")
		code := resp.StatusCode
		if code != 204 {
			t.Errorf("Incorrect response code, got %v", code)
		}
	})
}
