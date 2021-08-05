package auth_source

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"time"

	. "github.com/onsi/gomega"
	"gopkg.in/square/go-jose.v2"
	sqjwt "gopkg.in/square/go-jose.v2/jwt"
)

const ecdsaPrivKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKfldwWLPYsHjRL9EVTsjSbzTtcGRu6icohNfIqcb6A+oAoGCCqGSM49
AwEHoUQDQgAE4+SFvPwOy0miy/FiTT05HnwjpEbSq+7+1q9BFxAkzjgKnlkXk5qx
hzXQvRmS4w9ZsskoTZtuUI+XX7conJhzCQ==
-----END EC PRIVATE KEY-----`

const JWTPubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE4+SFvPwOy0miy/FiTT05HnwjpEbS
q+7+1q9BFxAkzjgKnlkXk5qxhzXQvRmS4w9ZsskoTZtuUI+XX7conJhzCQ==
-----END PUBLIC KEY-----`

const Audience = "https://flant.negentropy.com"

const issuer = "http://vault.example.com/"

func SignJWT(subject string, expireUnix time.Time, privateCl interface{}) string {
	var key *ecdsa.PrivateKey
	block, _ := pem.Decode([]byte(ecdsaPrivKey))
	if block != nil {
		var err error
		key, err = x509.ParseECPrivateKey(block.Bytes)
		Expect(err).ToNot(HaveOccurred())
	}

	cl := sqjwt.Claims{
		Subject:   subject,
		Issuer:    issuer,
		NotBefore: sqjwt.NewNumericDate(time.Now().Add(-5 * time.Second)),
		Audience:  sqjwt.Audience{Audience},
		Expiry:    sqjwt.NewNumericDate(expireUnix),
	}

	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	Expect(err).ToNot(HaveOccurred())

	raw, err := sqjwt.Signed(sig).Claims(cl).Claims(privateCl).CompactSerialize()
	Expect(err).ToNot(HaveOccurred())

	return raw
}
