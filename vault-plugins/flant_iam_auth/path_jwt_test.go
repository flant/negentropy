package jwtauth

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

func TestJWTConfigure(t *testing.T) {
	b, storage := getBackend(t)
	const jwtConfigurePath = "jwt/configure"

	// #1 Read the config
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      jwtConfigurePath,
		Storage:   storage,
		Data:      nil,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	require.Equal(t, false, resp.Data["enabled"])
	require.JSONEq(t, `{
"issuer":"https://auth.negentropy.flant.com/", 
"own_audience":"limbo", 
"preliminary_announce_period":"1d", 
"rotation_period":"1d"
}
`, resp.Data["configuration"].(string))

	// #2 Configure non default values
	req = &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      jwtConfigurePath,
		Storage:   storage,
		Data: map[string]interface{}{
			"issuer":                      "https://test",
			"own_audience":                "test",
			"preliminary_announce_period": "1h",
			"rotation_period":             "1h",
		},
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}

	// #3 Check again
	req = &logical.Request{
		Operation: logical.ReadOperation,
		Path:      jwtConfigurePath,
		Storage:   storage,
		Data:      nil,
	}

	resp, err = b.HandleRequest(context.Background(), req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%s resp:%#v\n", err, resp)
	}
	require.Equal(t, false, resp.Data["enabled"])
	require.JSONEq(t, `{
"issuer":"https://test", 
"own_audience":"test", 
"preliminary_announce_period":"1h", 
"rotation_period":"1h"
}
`, resp.Data["configuration"].(string))
}
