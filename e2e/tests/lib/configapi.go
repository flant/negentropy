package lib

import (
	"net/http"
)

// currently there are no needs to implement real ConfigAPI, as all configs provided by start.sh

type ConfigAPI interface {
	EnableJWT()
	GenerateCSR()
	ConfigureKafka(certificate string, kafkaEndpoints []string)
	ConfigureExtensionServerAccess(params map[string]interface{})
}

type httpClientBasedConfigAPI struct{}

func (h httpClientBasedConfigAPI) EnableJWT() {
}

func (h httpClientBasedConfigAPI) GenerateCSR() {
}

func (h httpClientBasedConfigAPI) ConfigureKafka(certificate string, kafkaEndpoints []string) {
}

func (h httpClientBasedConfigAPI) ConfigureExtensionServerAccess(params map[string]interface{}) {
}

func NewHttpClientBasedConfigAPI(_ *http.Client) ConfigAPI {
	return &httpClientBasedConfigAPI{}
}
