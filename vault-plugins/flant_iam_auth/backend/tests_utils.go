package backend

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	authn2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func getBackend(t *testing.T) (*flantIamAuthBackend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),

		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}
	b, err := FactoryWithJwksIDGetter(context.Background(), config, func() (string, error) {
		return "test", nil
	})

	fb := b.(*flantIamAuthBackend)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	entityIdResolver := mockEntityIDResolver{}
	fb.entityIDResolver = entityIdResolver
	fb.serverAccessBackend.SetEntityIDResolver(entityIdResolver)

	return fb, config.StorageView
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

func convertResponseToListKeys(t *testing.T, resp *api.Response) []string {
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

func extractResponseData(t *testing.T, resp *api.Response) map[string]interface{} {
	respRaw := map[string]interface{}{}
	err := resp.DecodeJSON(&respRaw)
	if err != nil {
		t.Errorf("Do not unmarshal body: %v", err)
	}

	return respRaw["data"].(map[string]interface{})
}

func extractResponseDataT(t *testing.T, resp *api.Response, out interface{}) {
	d := extractResponseData(t, resp)
	s, err := json.Marshal(d)
	if err != nil {
		t.Errorf("can not convert to json for cast: %v", err)
	}

	err = json.Unmarshal(s, out)
	if err != nil {
		t.Errorf("can not convert cast: %v", err)
	}
}

type apiRequester interface {
	Get(name string) *api.Response
	Create(name string, params map[string]interface{}) *api.Response
	Update(name string, params map[string]interface{}) *api.Response
	Delete(name string) *api.Response
	ListKeys() ([]string, *api.Response)
}

type rawVaultApiRequester struct {
	t      *testing.T
	cl     *api.Client
	prefix string
}

func newVaultRequester(t *testing.T, prefix string) *rawVaultApiRequester {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Fatalf("can not get client: %v", err)
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		token = "root"
	}

	client.SetToken(token)

	return &rawVaultApiRequester{
		prefix: prefix,
		t:      t,
		cl:     client,
	}
}

func (r *rawVaultApiRequester) Create(name string, params map[string]interface{}) *api.Response {
	return r.Request("POST", name, params, nil)
}

func (r *rawVaultApiRequester) Get(name string) *api.Response {
	return r.Request("GET", name, nil, nil)
}

func (r *rawVaultApiRequester) Update(name string, params map[string]interface{}) *api.Response {
	return r.Request("POST", name, params, nil)
}

func (r *rawVaultApiRequester) Delete(name string) *api.Response {
	return r.Request("DELETE", name, nil, nil)
}

func (r *rawVaultApiRequester) ListKeys() ([]string, *api.Response) {
	req := r.newRequest("GET", "", nil, &url.Values{
		"list": []string{"true"},
	})

	resp, err := r.cl.RawRequest(req)
	if resp == nil {
		r.t.Fatalf("error wile send request %v", err)
	}

	if resp.StatusCode == 404 {
		return make([]string, 0), resp
	}

	keys := convertResponseToListKeys(r.t, resp)

	return keys, resp
}

func (r *rawVaultApiRequester) Request(method, name string, params map[string]interface{}, q *url.Values) *api.Response {
	req := r.newRequest(method, name, params, q)
	resp, err := r.cl.RawRequest(req)
	if resp == nil {
		r.t.Fatalf("error wile send request %v", err)
	}

	return resp
}

func (r *rawVaultApiRequester) newRequest(method, name string, params map[string]interface{}, q *url.Values) *api.Request {
	path := fmt.Sprintf("/v1/auth/flant/%s/%s", r.prefix, name)
	request := r.cl.NewRequest(method, path)
	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			r.t.Fatalf("cannot marshal request params to json: %v", err)
			return nil
		}

		reader := bytes.NewReader(raw)
		request.Body = reader
	}

	if q != nil {
		request.Params = *q
	}

	return request
}

func assertResponseCode(t *testing.T, r *api.Response, code int) {
	rCode := r.StatusCode
	if code != rCode {
		t.Errorf("Incorrect response code, got %v; need %v", rCode, code)
	}
}

type mockEntityIDResolver struct{}

func (m mockEntityIDResolver) RevealEntityIDOwner(_ authn2.EntityID, _ *io.MemoryStoreTxn, _ logical.Storage) (*authn2.EntityIDOwner, error) {
	panic("if now need RevealEntityIDOwner, implement it")
}

func (m mockEntityIDResolver) AvailableTenantsByEntityID(_ authn2.EntityID, txn *io.MemoryStoreTxn, _ logical.Storage) (map[model.TenantUUID]struct{}, error) {
	tenantRepo := repo.NewTenantRepository(txn)
	tenants, err := tenantRepo.List(false)
	if err != nil {
		return nil, err
	}
	result := map[model.TenantUUID]struct{}{}
	for _, t := range tenants {
		result[t.UUID] = struct{}{}
	}
	return result, nil
}

func (m mockEntityIDResolver) AvailableProjectsByEntityID(_ authn2.EntityID, txn *io.MemoryStoreTxn, _ logical.Storage) (map[model.ProjectUUID]struct{}, error) {
	projectRepo := repo.NewProjectRepository(txn)
	tenantRepo := repo.NewTenantRepository(txn)
	tenants, err := tenantRepo.List(false)
	if err != nil {
		return nil, err
	}
	result := map[model.ProjectUUID]struct{}{}
	for _, t := range tenants {
		projects, err := projectRepo.List(t.UUID, false)
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			result[p.UUID] = struct{}{}
		}
	}
	return result, nil
}
