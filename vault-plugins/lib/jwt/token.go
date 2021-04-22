package jwt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"gopkg.in/square/go-jose.v2"
)

type PrimaryTokenClaims struct {
	Issuer   string `json:"iss"`
	Subject  string `json:"sub"`
	Audience string `json:"aud"` // TODO: can be array
	Expiry   int64  `json:"exp"`
	IssuedAt int64  `json:"iat"`
}

// newPrimaryToken is an example of token issuing function
func newPrimaryToken(ctx context.Context, storage logical.Storage) (string, error) {
	data, err := getConfig(ctx, storage)
	if err != nil {
		return "", err
	}

	issuer := data["issuer"].(string)
	audience := data["own_audience"].(string)

	entryPrivs, err := storage.Get(ctx, "jwt/private_keys")
	if err != nil {
		return "", err
	}

	keysSet := jose.JSONWebKeySet{}
	if len(entryPrivs.Value) > 0 {
		if err := json.Unmarshal(entryPrivs.Value, &keysSet); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("possible bug, keys not found in the storage")
	}

	issuedAt := time.Now()
	expiry := issuedAt.Add(time.Minute * 10)

	claims := PrimaryTokenClaims{
		Issuer:   issuer,
		Audience: audience,
		Subject:  "entropy",
		IssuedAt: issuedAt.Unix(),
		Expiry:   expiry.Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	firstKey := keysSet.Keys[0]

	// Hardcode alg here because we only support ed25519 keys
	token, err := signPayload(&firstKey, jose.EdDSA, payload)
	if err != nil {
		return "", err
	}
	return token, nil
}

// signPayload signs token
func signPayload(key *jose.JSONWebKey, alg jose.SignatureAlgorithm, payload []byte) (jws string, err error) {
	signingKey := jose.SigningKey{Key: key, Algorithm: alg}

	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return "", fmt.Errorf("new signer: %v", err)
	}
	signature, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("signing payload: %v", err)
	}
	return signature.CompactSerialize()
}
