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

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/featureflag"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tenant"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
)

type TestAPI interface {
	Create(tools.Params, url.Values, interface{})
	CreatePrivileged(tools.Params, url.Values, interface{})
	Read(tools.Params, url.Values)
	Update(tools.Params, url.Values, interface{})
	Delete(tools.Params, url.Values)
	List(tools.Params, url.Values)
}

type URLBuilder interface {
	One(tools.Params, url.Values) string
	Collection(tools.Params, url.Values) string
	Privileged(tools.Params, url.Values) string
}

var (
	_ TestAPI = (*BuilderBasedAPI)(nil)

	_ URLBuilder = (*tenant.EndpointBuilder)(nil)
	_ URLBuilder = (*featureflag.EndpointBuilder)(nil)
)

type BuilderBasedAPI struct {
	url    URLBuilder
	client *http.Client
}

func (b *BuilderBasedAPI) request(method, url string, params tools.Params, payload interface{}) {
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

	By(resp.Status+" | Payload: "+string(data), func() {
		if expectStatus, ok := params["expectStatus"]; ok {
			expectStatus.(func(response *http.Response))(resp)
		}

		if expectPayload, ok := params["expectPayload"]; ok {
			expectPayload.(func([]byte))(data)
		}
	})
}

func (b *BuilderBasedAPI) Create(params tools.Params, query url.Values, payload interface{}) {
	params.AddIfNotExists("expectStatus", tools.ExpectExactStatus(201))
	b.request(http.MethodPost, b.url.Collection(params, query), params, payload)
}

func (b *BuilderBasedAPI) CreatePrivileged(params tools.Params, query url.Values, payload interface{}) {
	params.AddIfNotExists("expectStatus", tools.ExpectExactStatus(201))
	b.request(http.MethodPost, b.url.Privileged(params, query), params, payload)
}

func (b *BuilderBasedAPI) Read(params tools.Params, query url.Values) {
	params.AddIfNotExists("expectStatus", tools.ExpectExactStatus(200))
	b.request(http.MethodGet, b.url.One(params, query), params, nil)
}

func (b *BuilderBasedAPI) Update(params tools.Params, query url.Values, payload interface{}) {
	params.AddIfNotExists("expectStatus", tools.ExpectExactStatus(200))
	b.request(http.MethodPost, b.url.One(params, query), params, payload)
}

func (b *BuilderBasedAPI) Delete(params tools.Params, query url.Values) {
	params.AddIfNotExists("expectStatus", tools.ExpectExactStatus(204))
	b.request(http.MethodDelete, b.url.One(params, query), params, nil)
}

func (b *BuilderBasedAPI) List(params tools.Params, query url.Values) {
	query.Set("list", "true")
	params.AddIfNotExists("expectStatus", tools.ExpectExactStatus(200))
	b.request(http.MethodGet, b.url.Collection(params, query), params, nil)
}

func NewTenantAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &tenant.EndpointBuilder{}}
}

func NewFeatureFlagAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &featureflag.EndpointBuilder{}}
}
