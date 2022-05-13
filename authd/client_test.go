package authd

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
)

// Warning! This test requires a negentropy started with ./start.sh (it rewrites authd/dev/secret/authd.jwt)
// and the authd process started with:
// from authd folder:
// go run cmd/authd/main.go --conf-dir=dev/conf
func Test_Client_LoginAndUseToken(t *testing.T) {
	// Socket path is from ./dev/conf/sock1.yaml
	authdClient := NewAuthdClient("./dev/run/sock1.sock")
	// "remote.example.com" is a random claim.
	req := v1.NewLoginRequest().
		WithRoles(v1.NewRoleWithClaim("ssh.open", map[string]interface{}{"max_ttl": "1000m", "ttl": "300s"})).
		WithServerType(v1.AuthServer)
	err := authdClient.OpenVaultSession(req)
	if err != nil {
		t.Fatalf("Should open vault session with authd library: %v", err)
	}

	vaultClient, err := authdClient.NewVaultClient()
	if err != nil {
		t.Fatalf("Should create new vault client with authd library: %v", err)
	}
	fmt.Printf("client token: %s\n", vaultClient.Token())

	// use authorized client
	s, err := vaultClient.Logical().List("auth/flant/tenant/")
	if err != nil {
		t.Fatalf("Should list tenants: %v", err)
	}
	if _, ok := s.Data["tenants"]; !ok {
		t.Fatalf("Should have tenants: %v", s.Data)
	}
}

func Test_Client_LoginAndRenewToken(t *testing.T) {
	// Socket path is from ./dev/conf/sock1.yaml
	authdClient := NewAuthdClient("./dev/run/sock1.sock")
	// "remote.example.com" is a random claim.
	req := v1.NewLoginRequest().
		WithRoles(v1.NewRoleWithClaim("ssh.open", map[string]interface{}{"max_ttl": "1000m", "ttl": "300s"})).
		WithServerType(v1.AuthServer)
	err := authdClient.OpenVaultSession(req)
	if err != nil {
		t.Fatalf("Should open vault session with authd library: %v", err)
	}
	vaultClient, err := authdClient.NewVaultClient()
	if err != nil {
		t.Fatalf("Should create new vault client with authd library: %v", err)
	}
	fmt.Printf("client token: %s\n", vaultClient.Token())
	// use authorized client
	_, err = vaultClient.Logical().List("auth/flant/tenant/")
	if err != nil {
		t.Fatalf("Should list tenants: %v", err)
	}

	refreshTime = time.Second
	authdClient.StartTokenRefresher(context.Background())
	time.Sleep(time.Second * 5)

	vaultClient, err = authdClient.NewVaultClient()
	if err != nil {
		t.Fatalf("Should create new vault client with authd library: %v", err)
	}
	// use client with refreshed token
	a, err := vaultClient.Auth().Token().LookupSelf()
	if err != nil {
		t.Fatalf("Should self_lookup token: %v", err)
	}
	ttl, err := a.TokenTTL()
	if err != nil {
		t.Fatalf("Should provide token ttl: %v", err)
	}
	// refresh set ttl to 10 minutes
	if ttl.Seconds() > 601.0 || ttl.Seconds() < 590.0 {
		t.Fatalf("ttl should be near 600s: %d", int(ttl.Seconds()))
	}
}
