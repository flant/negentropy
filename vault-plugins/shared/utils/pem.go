package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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
