package usecase

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
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
	return utils.ShaEncode(fmt.Sprintf("%d %s", j.Generation, j.SecretSalt))
}

type PrimaryTokenOptions struct {
	TTL  time.Duration
	UUID string
	JTI  TokenJTI

}

type TokenOptions struct {
	TTL time.Duration
	now func() time.Time
}

type TokenIssuer struct {
	config     *model.Config
	privateKey *jose.JSONWebKey
	now        func() time.Time
}

func NewTokenIssuer(config *model.Config, privateKey *jose.JSONWebKey, now func() time.Time) *TokenIssuer{
	return &TokenIssuer{
		config:     config,
		privateKey: privateKey,
		now:        now,
	}
}

func (s *TokenIssuer) PrimaryToken(options *PrimaryTokenOptions) (string, error) {
	issuedAt := s.now()
	expiry := issuedAt.Add(options.TTL)

	claims := PrimaryTokenClaims{
		TokenClaims: TokenClaims{
			Issuer:   s.config.Issuer,
			IssuedAt: issuedAt.Unix(),
			Expiry:   expiry.Unix(),
		},
		Audience: s.config.OwnAudience,
		Subject:  options.UUID,
		JTI:      options.JTI.Hash(),
	}

	return s.issue(claims)
}

func (s *TokenIssuer) Token(payload map[string]interface{}, options *TokenOptions) (string, error) {
	issuedAt := s.now()
	expiry := issuedAt.Add(options.TTL)

	claims, err := deepCopyMap(payload)
	if err != nil {
		return "", err
	}

	tokenClaims := TokenClaims{
		Issuer:   s.config.Issuer,
		IssuedAt: issuedAt.Unix(),
		Expiry:   expiry.Unix(),
	}

	err = addTokenClaimsToPayload(claims, &tokenClaims)
	if err != nil {
		return "", err
	}

	return s.issue(claims)
}

func (s *TokenIssuer) issue (payload interface{}) (string, error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Hardcode alg here because we only support ed25519 keys
	token, err := signPayload(s.privateKey, jose.EdDSA, payloadJson)
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
