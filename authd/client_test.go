package authd

import (
	"fmt"
	"testing"

	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
)

// Warning! This test requires a vault server started with dev/vault.sh
// and the authd process started with dev/start.sh.
func Test_Client_LoginAndUseTokenToReceiveOTPForSsh(t *testing.T) {
	// Socket path is from ./dev/conf/sock1.yaml
	authdClient := NewAuthdClient("./dev/run/sock1.sock")

	// "host" is a random claim.
	req := v1.NewLoginRequest().
		WithRoles(v1.NewRoleWithClaim("ssh.creds", map[string]string{"host": "remote.example.com"})).
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

	// Do some vault stuff...
	// The code below is not related to authd. This is just an example of simple vault client.
	// Use 127.0.0.1 as ssh ip. This IP should be within "cidr_list" used for otp_key_role.
	opts := map[string]interface{}{
		"ip": "127.0.0.1",
	}
	secret, err := vaultClient.SSH().Credential("otp_key_role", opts)
	if err != nil {
		t.Fatalf("vaultClient should be able to work with requested server: %v", err)
	}
	otp_key, ok := secret.Data["key"]
	if !ok {
		t.Fatalf("vaultClient should return otp for ssh in data['key']")
	}
	fmt.Printf("ssh otp: %s\n", otp_key)
}
