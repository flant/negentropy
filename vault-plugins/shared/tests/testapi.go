package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"go/token"
	"go/types"
	"net/url"
	"strings"

	"github.com/hashicorp/vault/sdk/logical"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

type Params = map[string]interface{}

type TestAPI interface {
	Create(Params, url.Values, interface{}) gjson.Result
	CreatePrivileged(Params, url.Values, interface{}) gjson.Result
	Read(Params, url.Values) gjson.Result
	Update(Params, url.Values, interface{}) gjson.Result
	Delete(Params, url.Values)
	List(Params, url.Values) gjson.Result
	Restore(Params, url.Values) gjson.Result
}

type PathBuilder interface {
	OneCreate(Params, url.Values) string
	One(Params, url.Values) string
	Collection(Params, url.Values) string
	Privileged(Params, url.Values) string
}

type BackendBasedAPI struct {
	Url     PathBuilder
	Backend *logical.Backend
	Storage *logical.Storage
}

func (b *BackendBasedAPI) request(operation logical.Operation, url string, params Params, payload interface{}) gjson.Result {
	p, ok := payload.(map[string]interface{})
	if !(operation == logical.ReadOperation || operation == logical.DeleteOperation || operation == logical.ListOperation) {
		Expect(ok).To(Equal(true), "definitely need map[string]interface{}")
	}
	url = strings.TrimSuffix(url, "?")

	request := &logical.Request{
		Operation: operation,
		Path:      url,
		Data:      p,
	}

	if b.Storage != nil {
		request.Storage = *b.Storage
	}

	resp, requestErr := (*b.Backend).HandleRequest(context.Background(), request)

	if requestErr == nil {
		if errRaw, gotErrMsg := resp.Data["error"]; gotErrMsg {
			errMsg := errRaw.(string)
			if strings.HasSuffix(errMsg, " field does not match the formatting rules") {
				requestErr = fmt.Errorf("%w: %s", consts.ErrInvalidArg, errMsg)
			} else {
				requestErr = fmt.Errorf("%s", errMsg)
			}
		}
	}

	if requestErr != nil {
		statusCodeInt := backentutils.MapErrorToHTTPStatusCode(requestErr)
		if statusCodeInt == 0 {
			statusCodeInt = 500
		}

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

func (b *BackendBasedAPI) Create(params Params, query url.Values, payload interface{}) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(201))
	return b.request(logical.CreateOperation, b.Url.OneCreate(params, query), params, payload)
}

func (b *BackendBasedAPI) CreatePrivileged(params Params, query url.Values, payload interface{}) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(201))
	return b.request(logical.CreateOperation, b.Url.Privileged(params, query), params, payload)
}

func (b *BackendBasedAPI) Read(params Params, query url.Values) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(200))
	return b.request(logical.ReadOperation, b.Url.One(params, query), params, nil)
}

func (b *BackendBasedAPI) Update(params Params, query url.Values, payload interface{}) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(200))
	return b.request(logical.UpdateOperation, b.Url.One(params, query), params, payload)
}

func (b *BackendBasedAPI) Delete(params Params, query url.Values) {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(204))
	b.request(logical.DeleteOperation, b.Url.One(params, query), params, nil)
}

func (b *BackendBasedAPI) List(params Params, query url.Values) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(200))
	var payload map[string]interface{} = nil
	if value, ok := query["show_archived"]; ok && value[0] == "true" {
		payload = map[string]interface{}{"show_archived": true}
		delete(query, "show_archived")
	}
	if value, ok := query["show_shared"]; ok {
		v := false
		if value[0] == "true" {
			v = true
		}
		if len(payload) == 0 {
			payload = map[string]interface{}{}
		}
		payload["show_shared"] = v
		delete(query, "show_shared")
	}

	return b.request(logical.ReadOperation, b.Url.Collection(params, query), params, payload)
}

func (b *BackendBasedAPI) Restore(params Params, query url.Values) gjson.Result {
	addIfNotExists(&params, "expectStatus", ExpectExactStatus(200))
	oneUrlWithParams := b.Url.One(params, query)
	parts := strings.Split(oneUrlWithParams, "?")
	url := parts[0] + "/restore"
	if len(parts) > 1 && parts[1] != "" {
		url = url + "?" + query.Encode()
	}
	emptyPayload := map[string]interface{}{}
	return b.request(logical.UpdateOperation, url, params, emptyPayload)
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
