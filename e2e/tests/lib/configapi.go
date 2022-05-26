package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

// currently there are no needs to implement some methods ConfigAPI, as some configs provided by start.sh

type ConfigAPI interface {
	EnableJWT()
	GenerateCSR()
	ConfigureKafka(certificate string, kafkaEndpoints []string)
	ConfigureExtensionServerAccess(params map[string]interface{})
	ConfigureExtensionFlantFlowFlantTenantUUID(flantTenantUUID model.TenantUUID)
	ConfigureExtensionFlantFlowAllFlantGroupUUID(allFlantGroupUUID model.GroupUUID)
	ConfigureExtensionFlantFlowAllFlantGroupRoles(allFlantGroupRoles []model.RoleName)
	ConfigureExtensionFlantFlowRoleRules(roles map[string][]string)
	ConfigureExtensionFlantFlowSpecificTeams(teams map[string]string)
	ReadConfigFlantFlow() config.FlantFlowConfig
	ConfigureExtensionFlantFlowClientPrimaryAdminsRoles(adminRoles []model.RoleName)
}

type httpClientBasedConfigAPI struct {
	httpVaultClient *http.Client
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowAllFlantGroupUUID(allFlantGroupUUID model.GroupUUID) {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowAllFlantGroupRoles(allFlantGroupRoles []model.RoleName) {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowFlantTenantUUID(flantTenantUUID model.TenantUUID) {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowRoleRules(rules map[string][]string) {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowSpecificTeams(teams map[string]string) {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowClientPrimaryAdminsRoles(adminRoles []model.RoleName) {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) ReadConfigFlantFlow() config.FlantFlowConfig {
	resp := h.request("GET", "/configure_extension/flant_flow", []int{http.StatusOK}, map[string]interface{}{})
	cfgRaw := resp.Get("flant_flow_cfg").String()
	var result config.FlantFlowConfig
	err := json.Unmarshal([]byte(cfgRaw), &result)
	Expect(err).ToNot(HaveOccurred())
	return result
}

func (h httpClientBasedConfigAPI) EnableJWT() {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) GenerateCSR() {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) ConfigureKafka(certificate string, kafkaEndpoints []string) {
	// start.sh & migrations
}

func (h httpClientBasedConfigAPI) ConfigureExtensionServerAccess(params map[string]interface{}) {
	// start.sh & migrations
}

func NewHttpClientBasedConfigAPI(client *http.Client) ConfigAPI {
	return &httpClientBasedConfigAPI{httpVaultClient: client}
}

func (h *httpClientBasedConfigAPI) request(method, url string, expectedStatuses []int, payload interface{}) gjson.Result {
	var body io.Reader
	if payload != nil {
		marshalPayload, err := json.Marshal(payload)
		Expect(err).ToNot(HaveOccurred())

		body = bytes.NewReader(marshalPayload)
	}

	req, err := http.NewRequest(method, url, body)
	Expect(err).ToNot(HaveOccurred())

	resp, err := h.httpVaultClient.Do(req)
	Expect(err).ToNot(HaveOccurred())

	defer resp.Body.Close()
	err = fmt.Errorf("wrong response status code: actual:%d, shoud be in :%v", resp.StatusCode, expectedStatuses)
	for _, s := range expectedStatuses {
		if s == resp.StatusCode {
			err = nil
			break
		}
	}
	Expect(err).ToNot(HaveOccurred())

	data, err := ioutil.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())

	json := tools.UnmarshalVaultResponse(data)

	return json
}
