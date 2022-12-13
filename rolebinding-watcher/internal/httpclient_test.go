package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/rolebinding-watcher/pkg"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
)

func Test_HttpClient(t *testing.T) {
	testURL := "/negentropy/backdoor"
	userEffectiveRoles := pkg.UserEffectiveRoles{
		UserUUID: "00000000-0000-4000-A000-000000000001",
		RoleName: "RoleName1",
		Tenants: []authz.EffectiveRoleTenantResult{
			{
				TenantUUID:       "00000001-0000-4000-A000-000000000000",
				TenantIdentifier: "tenant1",
				TenantOptions:    map[string][]interface{}{},
				Projects: []authz.EffectiveRoleProjectResult{
					{
						ProjectUUID:       "00000000-0100-4000-A000-000000000000",
						ProjectIdentifier: "pr1",
						ProjectOptions:    map[string][]interface{}{"o1": {"data1"}},
						RequireMFA:        false,
						NeedApprovals:     false,
					}, {
						ProjectUUID:       "00000000-0300-4000-A000-000000000000",
						ProjectIdentifier: "pr3",
						ProjectOptions:    map[string][]interface{}{"o1": {"data1"}},
						RequireMFA:        false,
						NeedApprovals:     false,
					},
				},
			},
		},
	}

	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		require.Equal(t, req.URL.String(), testURL)
		// Send response to be tested
		body := req.Body
		defer body.Close()
		var actialUserEffectiveRoles pkg.UserEffectiveRoles

		err := json.NewDecoder(req.Body).Decode(&actialUserEffectiveRoles)
		require.NoError(t, err)

		require.Equal(t, userEffectiveRoles.UserUUID, actialUserEffectiveRoles.UserUUID)
		require.Equal(t, userEffectiveRoles.RoleName, actialUserEffectiveRoles.RoleName)
		require.Equal(t, len(userEffectiveRoles.Tenants), len(actialUserEffectiveRoles.Tenants))
		rw.WriteHeader(200)
	}))

	defer server.Close()

	c := HTTPClient{
		Client: server.Client(),
		URL:    server.URL + testURL,
	}
	err := c.ProceedUserEffectiveRole(userEffectiveRoles)

	require.NoError(t, err)
}
