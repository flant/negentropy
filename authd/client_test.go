package authd

import (
	"fmt"
	"testing"

	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
)

func Test_Client_Login(t *testing.T) {
	authdClient := new(Client)
	authdClient.SocketPath = "./dev/run/sock1.sock"

	req := v1.NewLoginRequest().
		WithPolicies(v1.NewPolicy("ssh.creds", map[string]string{"host": "remote.example.com"})).
		WithServerType(v1.AuthServer)

	vaultClient, err := authdClient.Login(req)
	if err != nil {
		t.Fatalf("Should get vaultClient from authd library: %v", err)
	}

	fmt.Printf("client token: %s\n", vaultClient.Token())

	opts := map[string]interface{}{
		"ip": "127.0.0.1",
	}
	secret, err := vaultClient.SSH().Credential("otp_key_role", opts)
	if err != nil {
		t.Fatalf("vaultClient should be able to work with requested server: %v", err)
	}
	_, ok := secret.Data["key"]
	if !ok {
		t.Fatalf("vaultClient should return otp for ssh in data['key']")
	}
}
