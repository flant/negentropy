package api

import (
	"context"
	"strings"

	"github.com/hashicorp/vault/sdk/logical"
	. "github.com/onsi/gomega"
)

type ConfigAPI interface {
	EnableJWT()
	GenerateCSR()
	ConfigureKafka(certificate string, kafkaEndpoints []string)
	ConfigureExtensionServerAccess(params map[string]interface{})
}

type backendBasedConfigAPI struct {
	backend *logical.Backend
	storage *logical.Storage
}

func (b *backendBasedConfigAPI) GenerateCSR() {
	_, err := b.request(logical.CreateOperation, "kafka/generate_csr",
		map[string]interface{}{},
		map[string]interface{}{})
	Expect(err).ToNot(HaveOccurred())
}

func (b *backendBasedConfigAPI) ConfigureKafka(certificate string, kafkaEndpoints []string) {
	_, err := b.request(logical.UpdateOperation, "kafka/configure_access",
		map[string]interface{}{},
		map[string]interface{}{"kafka_endpoints": kafkaEndpoints})
	Expect(err).ToNot(HaveOccurred())
}

func (b *backendBasedConfigAPI) EnableJWT() {
	_, err := b.request(logical.UpdateOperation, "jwt/enable",
		map[string]interface{}{},
		map[string]interface{}{})
	Expect(err).ToNot(HaveOccurred())
}

func (b *backendBasedConfigAPI) ConfigureExtensionServerAccess(params map[string]interface{}) {
	_, err := b.request(logical.UpdateOperation, "configure_extension/server_access",
		map[string]interface{}{},
		params)
	Expect(err).ToNot(HaveOccurred())
}

func NewBackendBasedConfigAPI(backend *logical.Backend, storage *logical.Storage) ConfigAPI {
	return &backendBasedConfigAPI{
		backend: backend,
		storage: storage,
	}
}

func (b *backendBasedConfigAPI) request(operation logical.Operation, url string,
	params Params, payload interface{}) (map[string]interface{}, error) {
	p, ok := payload.(map[string]interface{})
	if !(operation == logical.ReadOperation || operation == logical.DeleteOperation || operation == logical.ListOperation) {
		Expect(ok).To(Equal(true), "definitely need map[string]interface{}")
	}
	url = strings.TrimSuffix(url, "?")
	resp, err := (*b.backend).HandleRequest(context.Background(), &logical.Request{
		Operation: operation,
		Path:      url,
		Data:      p,
		Storage:   *b.storage,
	})
	if resp == nil {
		return nil, err
	}
	return resp.Data, err
}
