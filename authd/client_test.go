package authd

import (
	"fmt"
	"testing"

	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
)

// Warning! This test requires a negentropy started with ./start.sh (it rewrites authd/dev/secret/authd.jwt)
// and the authd process started with:
// from authd folder:
// go run cmd/authd/main.go --conf-dir=dev/conf
func Test_Client_LoginAndUseTokenToReceiveOTPForSsh(t *testing.T) {
	// Socket path is from ./dev/conf/sock1.yaml
	authdClient := NewAuthdClient("./dev/run/sock1.sock")

	// "remote.example.com" is a random claim.
	req := v1.NewLoginRequest().
		WithRoles(v1.NewRoleWithClaim("ssh", "remote.example.com")).
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

	s, err := vaultClient.Logical().List("auth/flant_iam_auth/tenant/")

	if err != nil {
		t.Fatalf("Should list tenants: %v", err)
	}

	if _, ok := s.Data["tenants"]; !ok {
		t.Fatalf("Should have tenants: %v", s.Data)
	}
}
