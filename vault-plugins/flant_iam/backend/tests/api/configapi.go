package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/logical"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

type ConfigAPI interface {
	EnableJWT()
	ConfigureKafka(certificate string, kafkaEndpoints []string)
	ConfigureExtensionServerAccess(params map[string]interface{})
	ConfigureExtensionFlantFlowFlantTenantUUID(flantTenantUUID model.TenantUUID)
	ConfigureExtensionFlantFlowAllFlantGroupUUID(allFlantGroupUUID model.GroupUUID)
	ConfigureExtensionFlantFlowAllFlantGroupRoles(allFlantGroupRoles []model.RoleName)
	ConfigureExtensionServicePacksRolesSpecification(specification config.ServicePacksRolesSpecification)
	ConfigureExtensionFlantFlowSpecificTeams(teams map[string]string)
	ConfigureExtensionFlantFlowClientPrimaryAdminsRoles(adminRoles []model.RoleName)
	ReadConfigFlantFlow() config.FlantFlowConfig
}

type backendBasedConfigAPI struct {
	backend *logical.Backend
	storage *logical.Storage
}

func (b *backendBasedConfigAPI) ConfigureExtensionFlantFlowClientPrimaryAdminsRoles(adminRoles []model.RoleName) {
	resp, err := b.request(logical.CreateOperation, "configure_extension/flant_flow/client_primary_administrators_roles",
		map[string]interface{}{},
		map[string]interface{}{"roles": adminRoles})
	Expect(err).ToNot(HaveOccurred())
	if bodyStr, ok := resp["http_raw_body"].(string); ok {
		valid := gjson.Valid(bodyStr)
		Expect(valid).To(BeTrue())
		body := gjson.Parse(bodyStr)
		if errMsg := body.Get("data.error").String(); errMsg != "" {
			err = fmt.Errorf(errMsg)
			Expect(err).ToNot(HaveOccurred())
		}
	}
}

func (b *backendBasedConfigAPI) ConfigureExtensionFlantFlowAllFlantGroupRoles(allFlantGroupRoles []model.RoleName) {
	resp, err := b.request(logical.CreateOperation, "configure_extension/flant_flow/all_flant_group_roles",
		map[string]interface{}{},
		map[string]interface{}{"roles": allFlantGroupRoles})
	Expect(err).ToNot(HaveOccurred())
	if bodyStr, ok := resp["http_raw_body"].(string); ok {
		valid := gjson.Valid(bodyStr)
		Expect(valid).To(BeTrue())
		body := gjson.Parse(bodyStr)
		if errMsg := body.Get("data.error").String(); errMsg != "" {
			err = fmt.Errorf(errMsg)
			Expect(err).ToNot(HaveOccurred())
		}
	}
}

func (b *backendBasedConfigAPI) ReadConfigFlantFlow() config.FlantFlowConfig {
	resp, err := b.request(logical.ReadOperation, "configure_extension/flant_flow", map[string]interface{}{}, map[string]interface{}{})
	Expect(err).ToNot(HaveOccurred())
	js := gjson.Parse(resp["http_raw_body"].(string))
	cfgRaw := js.Get("data.flant_flow_cfg").String()
	var result config.FlantFlowConfig
	err = json.Unmarshal([]byte(cfgRaw), &result)
	Expect(err).ToNot(HaveOccurred())
	return result
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

func (b *backendBasedConfigAPI) ConfigureExtensionFlantFlowFlantTenantUUID(flantTenantUUID model.TenantUUID) {
	_, err := b.request(logical.UpdateOperation, "configure_extension/flant_flow/flant_tenant/"+flantTenantUUID,
		map[string]interface{}{},
		map[string]interface{}{})
	Expect(err).ToNot(HaveOccurred())
}

func (b *backendBasedConfigAPI) ConfigureExtensionFlantFlowAllFlantGroupUUID(allFlantGroupUUID model.GroupUUID) {
	_, err := b.request(logical.UpdateOperation, "configure_extension/flant_flow/all_flant_group/"+allFlantGroupUUID,
		map[string]interface{}{},
		map[string]interface{}{})
	Expect(err).ToNot(HaveOccurred())
}

func (b *backendBasedConfigAPI) ConfigureExtensionServicePacksRolesSpecification(specification config.ServicePacksRolesSpecification) {
	_, err := b.request(logical.UpdateOperation, "configure_extension/flant_flow/service_packs_roles_specification",
		map[string]interface{}{},
		map[string]interface{}{"specification": specification})
	Expect(err).ToNot(HaveOccurred())
}

func (b *backendBasedConfigAPI) ConfigureExtensionFlantFlowSpecificTeams(teams map[string]string) {
	_, err := b.request(logical.UpdateOperation, "configure_extension/flant_flow/specific_teams",
		map[string]interface{}{},
		map[string]interface{}{"specific_teams": teams})
	Expect(err).ToNot(HaveOccurred())
}

func NewBackendBasedConfigAPI(backend *logical.Backend, storage *logical.Storage) ConfigAPI {
	return &backendBasedConfigAPI{
		backend: backend,
		storage: storage,
	}
}

func (b *backendBasedConfigAPI) request(operation logical.Operation, url string,
	_ tests.Params, payload interface{}) (map[string]interface{}, error) {
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
