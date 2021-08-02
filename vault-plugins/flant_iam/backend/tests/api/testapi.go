package api

import (
	"context"
	"encoding/json"
	"fmt"
	"go/token"
	"go/types"
	"net/url"
	"strings"

	url2 "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/url"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/sdk/physical/inmem"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend"
)

type Params = map[string]interface{}

type TestAPI interface {
	Create(Params, url.Values, interface{}) gjson.Result
	CreatePrivileged(Params, url.Values, interface{}) gjson.Result
	Read(Params, url.Values) gjson.Result
	Update(Params, url.Values, interface{}) gjson.Result
	Delete(Params, url.Values)
	List(Params, url.Values) gjson.Result
}

type PathBuilder interface {
	OneCreate(Params, url.Values) string
	One(Params, url.Values) string
	Collection(Params, url.Values) string
	Privileged(Params, url.Values) string
}

type BuilderBasedAPI struct {
	url     PathBuilder
	backend *logical.Backend
}

func (b *BuilderBasedAPI) request(operation logical.Operation, url string, params Params, payload interface{}) gjson.Result {
	p, ok := payload.(map[string]interface{})
	if !(operation == logical.ReadOperation || operation == logical.DeleteOperation || operation == logical.ListOperation) {
		Expect(ok).To(Equal(true), "definitely need map[string]interface{}")
	}
	if strings.HasSuffix(url, "?") {
		url = url[:len(url)-1]
	}
	resp, requestErr := (*b.backend).HandleRequest(context.Background(), &logical.Request{
		Operation: operation,
		Path:      url,
		Data:      p,
	})

	if requestErr != nil {
		statusCodeInt := 500
		By(fmt.Sprintf("%d | Payload: %v", statusCodeInt, payload),
			func() {
				if expectStatus, ok := params["expectStatus"]; ok {
					expectStatus.(func(int))(statusCodeInt)
				}
			})
		return gjson.Result{}
	}

	statusCode, ok := resp.Data["http_status_code"]
	Expect(ok).To(Equal(true), "definitely need http_status_code in vault response")

	statusCodeInt, ok := statusCode.(int)
	Expect(ok).To(Equal(true), "http_status_code should be int")

	json := gjson.Result{}

	if operation != logical.DeleteOperation {
		rawBody, ok := resp.Data["http_raw_body"]
		Expect(ok).To(Equal(true), "definitely need http_raw_body in vault response")

		body, ok := rawBody.(string)
		Expect(ok).To(Equal(true), "http_raw_body should be string")

		json = gjson.Parse(body).Get("data")
	}

	By(fmt.Sprintf("%d | Payload: %v", statusCodeInt, payload),
		func() {
			if expectStatus, ok := params["expectStatus"]; ok {
				expectStatus.(func(int))(statusCodeInt)
			}

			if expectPayload, ok := params["expectPayload"]; ok {
				expectPayload.(func(gjson.Result))(json)
			}
		})

	return json
}

func (b *BuilderBasedAPI) Create(params Params, query url.Values, payload interface{}) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(201))
	return b.request(logical.CreateOperation, b.url.OneCreate(params, query), params, payload)
}

func (b *BuilderBasedAPI) CreatePrivileged(params Params, query url.Values, payload interface{}) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(201))
	return b.request(logical.CreateOperation, b.url.Privileged(params, query), params, payload)
}

func (b *BuilderBasedAPI) Read(params Params, query url.Values) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(200))
	return b.request(logical.ReadOperation, b.url.One(params, query), params, nil)
}

func (b *BuilderBasedAPI) Update(params Params, query url.Values, payload interface{}) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(200))
	return b.request(logical.UpdateOperation, b.url.One(params, query), params, payload)
}

func (b *BuilderBasedAPI) Delete(params Params, query url.Values) {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(204))
	b.request(logical.DeleteOperation, b.url.One(params, query), params, nil)
}

func (b *BuilderBasedAPI) List(params Params, query url.Values) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(200))
	return b.request(logical.ReadOperation, b.url.Collection(params, query), params, nil)
}

func NewRoleAPI(b *logical.Backend) TestAPI {
	return &BuilderBasedAPI{backend: b, url: &url2.RoleEndpointBuilder{}}
}

func NewFeatureFlagAPI(b *logical.Backend) TestAPI {
	return &BuilderBasedAPI{backend: b, url: &url2.FeatureFlagEndpointBuilder{}}
}

func NewTenantAPI(b *logical.Backend) TestAPI {
	return &BuilderBasedAPI{backend: b, url: &url2.TenantEndpointBuilder{}}
}

func NewIdentitySharingAPI(b *logical.Backend) TestAPI {
	return &BuilderBasedAPI{backend: b, url: &url2.IdentitySharingEndpointBuilder{}}
}

func NewRoleBindingAPI(b *logical.Backend) TestAPI {
	return &BuilderBasedAPI{backend: b, url: &url2.RoleBindingEndpointBuilder{}}
}

func NewRoleBindingApprovalAPI(b *logical.Backend) TestAPI {
	return &BuilderBasedAPI{backend: b, url: &url2.RoleBindingApprovalEndpointBuilder{}}
}

func NewTenantFeatureFlagAPI(b *logical.Backend) TestAPI {
	return &BuilderBasedAPI{backend: b, url: &url2.TenantFeatureFlagEndpointBuilder{}}
}

func ExpectExactStatus(expectedStatus int) func(gotStatus int) {
	return func(gotStatus int) {
		Expect(gotStatus).To(Equal(expectedStatus))
	}
}

func ExpectStatus(condition string) func(gotStatus int) {
	return func(gotStatus int) {
		formula := fmt.Sprintf(condition, gotStatus)
		By("Status code check "+formula, func() {
			fs := token.NewFileSet()

			tv, err := types.Eval(fs, nil, token.NoPos, formula)
			Expect(err).ToNot(HaveOccurred())

			Expect(tv.Value.String()).To(Equal("true"))
		})
	}
}

type VaultPayload struct {
	Data json.RawMessage `json:"data"`
}

func ToMap(v interface{}) map[string]interface{} {
	js, err := json.Marshal(v)
	Expect(err).ToNot(HaveOccurred())
	out := map[string]interface{}{}
	err = json.Unmarshal(js, &out)
	Expect(err).ToNot(HaveOccurred())

	return out
}

func addIfNotExists(p *Params, key string, val interface{}) {
	if *p == nil {
		*p = make(map[string]interface{})
	}

	data := *p
	if _, ok := data[key]; !ok {
		data[key] = val
	}
}

func TestBackend() logical.Backend {
	config := logical.TestBackendConfig()
	testPhisicalBackend, _ := inmem.NewInmemHA(map[string]string{}, config.Logger)
	config.StorageView = logical.NewStorageView(logical.NewLogicalStorage(testPhisicalBackend), "")
	b, _ := backend.Factory(context.Background(), config)
	return b
}
