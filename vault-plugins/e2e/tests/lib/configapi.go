package lib

import (
	"net/http"
)

// currently there is no needs to implement real ConfigAPI

type ConfigAPI interface {
	EnableJWT()
	GenerateCSR()
	ConfigureKafka(certificate string, kafkaEndpoints []string)
}

type httpClientBasedConfigAPI struct{}

func (h httpClientBasedConfigAPI) EnableJWT() {
}

func (h httpClientBasedConfigAPI) GenerateCSR() {
}

func (h httpClientBasedConfigAPI) ConfigureKafka(certificate string, kafkaEndpoints []string) {
}

func NewHttpClientBasedConfigAPI(_ *http.Client) ConfigAPI {
	return &httpClientBasedConfigAPI{}
}
