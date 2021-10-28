package lib

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/e2e/tests/lib/tools"
	url2 "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/url"
)

type TestAPI interface {
	Create(tools.Params, url.Values, interface{}) gjson.Result
	CreatePrivileged(tools.Params, url.Values, interface{}) gjson.Result
	Read(tools.Params, url.Values) gjson.Result
	Update(tools.Params, url.Values, interface{}) gjson.Result
	Delete(tools.Params, url.Values)
	List(tools.Params, url.Values) gjson.Result
}

type URLBuilder interface {
	OneCreate(tools.Params, url.Values) string
	One(tools.Params, url.Values) string
	Collection(tools.Params, url.Values) string
	Privileged(tools.Params, url.Values) string
}

var (
	_ TestAPI = (*BuilderBasedAPI)(nil)

	_ URLBuilder = (*url2.TenantEndpointBuilder)(nil)
	_ URLBuilder = (*url2.FeatureFlagEndpointBuilder)(nil)
	_ URLBuilder = (*url2.IdentitySharingEndpointBuilder)(nil)
	_ URLBuilder = (*url2.RoleEndpointBuilder)(nil)
	_ URLBuilder = (*url2.RoleBindingApprovalEndpointBuilder)(nil)
	_ URLBuilder = (*url2.TenantFeatureFlagEndpointBuilder)(nil)
	_ URLBuilder = (*url2.RoleEndpointBuilder)(nil)
)

type BuilderBasedAPI struct {
	url    URLBuilder
	client *http.Client
}

func (b *BuilderBasedAPI) request(method, url string, params tools.Params, payload interface{}) gjson.Result {
	var body io.Reader
	if payload != nil {
		marshalPayload, err := json.Marshal(payload)
		Expect(err).ToNot(HaveOccurred())

		body = bytes.NewReader(marshalPayload)
	}

	req, err := http.NewRequest(method, url, body)
	Expect(err).ToNot(HaveOccurred())

	resp, err := b.client.Do(req)
	Expect(err).ToNot(HaveOccurred())

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())

	json := tools.UnmarshalVaultResponse(data)

	By(resp.Status+" | Payload: "+string(data), func() {
		if expectStatus, ok := params["expectStatus"]; ok {
			expectStatus.(func(statusCode int))(resp.StatusCode)
		}

		if expectPayload, ok := params["expectPayload"]; ok {
			expectPayload.(func(result gjson.Result))(json)
		}
	})

	return json
}

func (b *BuilderBasedAPI) Create(params tools.Params, query url.Values, payload interface{}) gjson.Result {
	tools.AddIfNotExists(&params, "expectStatus", tools.ExpectExactStatus(201))
	return b.request(http.MethodPost, "/"+b.url.OneCreate(params, query), params, payload)
}

func (b *BuilderBasedAPI) CreatePrivileged(params tools.Params, query url.Values, payload interface{}) gjson.Result {
	tools.AddIfNotExists(&params, "expectStatus", tools.ExpectExactStatus(201))
	return b.request(http.MethodPost, "/"+b.url.Privileged(params, query), params, payload)
}

func (b *BuilderBasedAPI) Read(params tools.Params, query url.Values) gjson.Result {
	tools.AddIfNotExists(&params, "expectStatus", tools.ExpectExactStatus(200))
	return b.request(http.MethodGet, "/"+b.url.One(params, query), params, nil)
}

func (b *BuilderBasedAPI) Update(params tools.Params, query url.Values, payload interface{}) gjson.Result {
	tools.AddIfNotExists(&params, "expectStatus", tools.ExpectExactStatus(200))
	return b.request(http.MethodPost, "/"+b.url.One(params, query), params, payload)
}

func (b *BuilderBasedAPI) Delete(params tools.Params, query url.Values) {
	tools.AddIfNotExists(&params, "expectStatus", tools.ExpectExactStatus(204))
	b.request(http.MethodDelete, "/"+b.url.One(params, query), params, nil)
}

func (b *BuilderBasedAPI) List(params tools.Params, query url.Values) gjson.Result {
	// query.Set("list", "true")
	tools.AddIfNotExists(&params, "expectStatus", tools.ExpectExactStatus(200))
	return b.request(http.MethodGet, "/"+b.url.Collection(params, query), params, nil)
}

func NewTenantAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.TenantEndpointBuilder{}}
}

func NewFeatureFlagAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.FeatureFlagEndpointBuilder{}}
}

func NewIdentitySharingAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.IdentitySharingEndpointBuilder{}}
}

func NewRoleBindingAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.RoleBindingEndpointBuilder{}}
}

func NewRoleBindingApprovalAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.RoleBindingApprovalEndpointBuilder{}}
}

func NewTenantFeatureFlagAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.TenantFeatureFlagEndpointBuilder{}}
}

func NewUserAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.UserEndpointBuilder{}}
}

func NewRoleAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.RoleEndpointBuilder{}}
}

func NewGroupAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.GroupEndpointBuilder{}}
}

func NewProjectAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ProjectEndpointBuilder{}}
}

func NewUserMultipassAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.UserMultipassEndpointBuilder{}}
}

func NewServerAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ServerEndpointBuilder{}}
}

func NewConnectionInfoAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ConnectionInfoEndpointBuilder{}}
}

func NewServiceAccountMultipassAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ServiceAccountMultipassEndpointBuilder{}}
}

func NewProjectFeatureFlagAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ProjectFeatureFlagEndpointBuilder{}}
}
