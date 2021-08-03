package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func DecodePemKey(key *rsa.PublicKey) string {
	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(key),
		},
	)

	return string(pemdata)
}

func ParsePubkey(data string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, errors.New("key can not be parsed")
	}
	pubkey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return pubkey, nil
}
