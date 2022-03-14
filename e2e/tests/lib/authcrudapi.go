package lib

import (
	"net/http"

	url2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/backend/tests/url"
)

func NewPolicyAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.PolicyEndpointBuilder{}}
}
