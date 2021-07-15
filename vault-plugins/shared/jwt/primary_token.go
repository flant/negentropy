package jwt

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"gopkg.in/square/go-jose.v2"
)

type TokenClaims struct {
	Issuer   string `json:"iss"`
	Expiry   int64  `json:"exp"`
	IssuedAt int64  `json:"iat"`
}

type PrimaryTokenClaims struct {
	TokenClaims
	Subject  string `json:"sub"`
	Audience string `json:"aud"` // TODO: can be array, but we have no case to make it array right now
	JTI      string `json:"jti"`
}

type TokenJTI struct {
	Generation int64
	SecretSalt string
}

func (j TokenJTI) Hash() string {
	return shaEncode(fmt.Sprintf("%d %s", j.Generation, j.SecretSalt))
}

type PrimaryTokenOptions struct {
	TTL  time.Duration
	UUID string
	JTI  TokenJTI

	now func() time.Time
}

func (o *PrimaryTokenOptions) getCurrentTime() time.Time {
	if o.now == nil {
		o.now = time.Now
	}
	return o.now()
}

type TokenOptions struct {
	TTL time.Duration
	now func() time.Time
}

func (o *TokenOptions) getCurrentTime() time.Time {
	if o.now == nil {
		o.now = time.Now
	}
	return o.now()
}

// NewPrimaryToken is tokens issuing function
func NewPrimaryToken(ctx context.Context, storage logical.Storage, options *PrimaryTokenOptions) (string, error) {
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

	issuedAt := options.getCurrentTime()
	expiry := issuedAt.Add(options.TTL)

	claims := PrimaryTokenClaims{
		TokenClaims: TokenClaims{
			Issuer:   issuer,
			IssuedAt: issuedAt.Unix(),
			Expiry:   expiry.Unix(),
		},
		Audience: audience,
		Subject:  options.UUID,
		JTI:      options.JTI.Hash(),
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

func NewJwtToken(ctx context.Context, storage logical.Storage, payload map[string]interface{}, options *TokenOptions) (string, error) {
	data, err := getConfig(ctx, storage)
	if err != nil {
		return "", err
	}

	issuer := data["issuer"].(string)

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

	issuedAt := options.getCurrentTime()
	expiry := issuedAt.Add(options.TTL)

	claims, err := deepCopyMap(payload)
	if err != nil {
		return "", err
	}

	tokenClaims := TokenClaims{
		Issuer:   issuer,
		IssuedAt: issuedAt.Unix(),
		Expiry:   expiry.Unix(),
	}

	err = addTokenClaimsToPayload(claims, &tokenClaims)
	if err != nil {
		return "", err
	}

	payloadJson, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	firstKey := keysSet.Keys[0]

	// Hardcode alg here because we only support ed25519 keys
	token, err := signPayload(&firstKey, jose.EdDSA, payloadJson)
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

func shaEncode(input string) string {
	// TODO: declare hasher once
	hasher := sha256.New()

	hasher.Write([]byte(input))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func deepCopyMap(obj map[string]interface{}) (map[string]interface{}, error) {
	jsonStr, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	dst := map[string]interface{}{}
	err = json.Unmarshal(jsonStr, &dst)
	if err != nil {
		return nil, err
	}

	return dst, nil
}

func addTokenClaimsToPayload(payload map[string]interface{}, claims *TokenClaims) error {
	jsonStr, err := json.Marshal(claims)
	if err != nil {
		return err
	}

	mapClaims := map[string]interface{}{}
	err = json.Unmarshal(jsonStr, &mapClaims)
	if err != nil {
		return err
	}

	for k, v := range mapClaims {
		payload[k] = v
	}

	return nil
}
