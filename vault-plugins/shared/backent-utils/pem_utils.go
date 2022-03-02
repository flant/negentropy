package backentutils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
)

func ConvertToPem(pk *rsa.PublicKey) string {
	if pk == nil {
		return ""
	}
	encoded := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(pk),
	})
	return strings.ReplaceAll(string(encoded), "\n", "\\n")
}

func ConvertToPems(pks []*rsa.PublicKey) []string {
	if len(pks) == 0 {
		return nil
	}
	result := make([]string, 0, len(pks))
	for _, pk := range pks {
		result = append(result, ConvertToPem(pk))
	}
	return result
}
