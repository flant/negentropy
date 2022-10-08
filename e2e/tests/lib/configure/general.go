package configure

import (
	"net/http"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/gomega"
)

// GetClientWithToken create vault client with customized token and addr
// example: GetClientWithToken("token","https://127.0.0.1:8200")
func GetClientWithToken(token string, addr string) *api.Client {
	cfg := api.DefaultConfig()
	transport := cfg.HttpClient.Transport.(*http.Transport)
	transport.TLSClientConfig.InsecureSkipVerify = true
	cl, err := api.NewClient(cfg)
	Expect(err).To(BeNil())

	cl.SetToken(token)
	err = cl.SetAddress(addr)
	Expect(err).To(BeNil())

	return cl
}

func CreatePolicy(vaultClient *api.Client, name, content string) {
	policyFromServer, err := vaultClient.Sys().GetPolicy(name)
	Expect(err).To(BeNil())

	if policyFromServer == "" {
		err := vaultClient.Sys().PutPolicy(name, content)
		Expect(err).To(BeNil())
	}
}
