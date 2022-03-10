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
	ConfigureExtensionFlantFlowRoleRules(roles map[string][]string)
	ConfigureExtensionFlantFlowSpecificTeams(teams map[string]string)
	ReadConfigFlantFlow() config.FlantFlowConfig
}

type httpClientBasedConfigAPI struct {
	httpVaultClient *http.Client
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowFlantTenantUUID(flantTenantUUID model.TenantUUID) {
	// by start.sh
	// h.request("POST", "/configure_extension/flant_flow/flant_tenant/"+flantTenantUUID, []int{http.StatusOK, http.StatusBadRequest}, nil)
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowRoleRules(rules map[string][]string) {
	// by start.sh
	// for team, roles := range rules {
	//	h.request("POST", "/configure_extension/flant_flow/role_rules/"+team, []int{http.StatusOK},
	//		map[string]interface{}{"specific_roles": roles})
	// }
}

func (h httpClientBasedConfigAPI) ConfigureExtensionFlantFlowSpecificTeams(teams map[string]string) {
	// by start.sh
	// h.request("POST", "/configure_extension/flant_flow/specific_teams", []int{http.StatusOK},
	//	map[string]interface{}{"specific_teams": teams})
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
	// by start.sh
}

func (h httpClientBasedConfigAPI) GenerateCSR() {
	// by start.sh
}

func (h httpClientBasedConfigAPI) ConfigureKafka(certificate string, kafkaEndpoints []string) {
	// by start.sh
}

func (h httpClientBasedConfigAPI) ConfigureExtensionServerAccess(params map[string]interface{}) {
	// by start.sh
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
