package vault

import (
	"context"
	"strings"
	"testing"

	"github.com/flant/negentropy/authd/pkg/jwt"
)

// Warning! This test requires a negentropy started with ./start.sh (it rewrites authd/dev/secret/authd.jwt)
func Test_LoginWithJWT(t *testing.T) {
	storage := jwt.Storage{Path: "../../dev/secret/authd.jwt"}
	multipassJWT, err := storage.GetJWT()
	if err != nil {
		t.Fatalf("reading multipass should be succesful. error (type %T): %v", err, err)
	}
	vcl := NewClient("http://127.0.0.1:8200")

	secret, err := vcl.LoginWithJWTAndClaims(context.Background(), multipassJWT, nil)

	if err != nil {
		t.Fatalf("login should be successful. error (type %T): %v", err, err)
	}
	if !strings.HasPrefix(secret.Auth.ClientToken, "s.") {
		t.Fatalf("client token should starts with s., got: '%s'", secret.Auth.ClientToken)
	}
}

// Warning! This test requires a negentropy started with ./start.sh (it rewrites authd/dev/secret/authd.jwt)
func TestClient_RefreshJWT(t *testing.T) {
	storage := jwt.Storage{Path: "../../dev/secret/authd.jwt"}
	multipassJWT, err := storage.GetJWT()
	if err != nil {
		t.Fatalf("reading multipass should be succesful. error (type %T): %v", err, err)
	}
	vcl := NewClient("http://127.0.0.1:8200")

	newMultipassJWT, err := vcl.RefreshJWT(context.TODO(), multipassJWT)

	if err != nil {
		t.Fatalf("RefreshJWT should be successful. error (type %T): %v", err, err)
	}
	oldMultipass, err := jwt.ParseToken(multipassJWT)
	if err != nil {
		t.Fatalf("ParseToken should be successful. error (type %T): %v", err, err)
	}
	newMultipass, err := jwt.ParseToken(newMultipassJWT)
	if err != nil {
		t.Fatalf("ParseToken should be successful. error (type %T): %v", err, err)
	}
	if oldMultipass.Payload["sub"] != newMultipass.Payload["sub"] {
		t.Fatalf("sub should be same, expected %v, got %v", oldMultipass.Payload["sub"], newMultipass.Payload["sub"])
	}
	if oldMultipass.Payload["iat"].(float64) >= newMultipass.Payload["iat"].(float64) {
		t.Fatalf("iat should became greater %v, new: %v", oldMultipass.Payload["iat"], newMultipass.Payload["iat"])
	}
	if oldMultipass.Payload["jti"] == newMultipass.Payload["jti"] {
		t.Fatalf("jti should not be same, old %v, new %v", oldMultipass.Payload["jti"], newMultipass.Payload["jti"])
	}

	// Store new multipass to repeat tests
	err = storage.Update(newMultipassJWT)
	if err != nil {
		t.Fatalf("Update should be successful. error (type %T): %v", err, err)
	}
}
