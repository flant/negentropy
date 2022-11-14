package kafka

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"strconv"
)

type Encrypter struct{}

var separator = strconv.QuoteRuneToASCII('☺')

// DoNotEncrypt specify doesn't use encrypting if true
var DoNotEncrypt bool

func NewEncrypter() *Encrypter {
	return &Encrypter{}
}

func (c *Encrypter) Encrypt(data []byte, pub *rsa.PublicKey) (env []byte, chunked bool, err error) {
	dataLen := len(data)
	if DoNotEncrypt {
		env = data
		return
	}

	hash := sha256.New()
	maxSize := pub.Size() - 2*hash.Size() - 2
	// The message must be no longer than the length of the public modulus minus
	// twice the hash length, minus a further 2.
	if dataLen <= maxSize { //  k-2*hash.Size()-2
		env, err = rsa.EncryptOAEP(hash, rand.Reader, pub, data, nil)
		return
	}

	buf := bytes.NewBuffer(nil)

	chunked = true
	parts := dataLen / maxSize

	for i := 0; i <= parts; i++ {
		var chunk []byte
		if i == parts {
			chunk = data[i*maxSize:]
		} else {
			chunk = data[i*maxSize : (i+1)*maxSize]
		}
		eChunk, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, chunk, nil)
		if err != nil {
			return nil, chunked, err
		}
		buf.Write(eChunk)
		if i != parts {
			buf.WriteString(separator)
		}
	}

	return buf.Bytes(), chunked, nil
}

func (c *Encrypter) Decrypt(data []byte, priv *rsa.PrivateKey, chunked bool) ([]byte, error) {
	if DoNotEncrypt || len(data) == 0 { // empty data decryption leads error "crypto/rsa: decryption error"
		return data, nil
	}
	if !chunked {
		return rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, data, nil)
	}
	buf := bytes.NewBuffer(nil)

	chunks := bytes.Split(data, []byte(separator))

	for _, chunk := range chunks {
		dec, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, chunk, nil)
		if err != nil {
			return nil, err
		}
		buf.Write(dec)
	}

	return buf.Bytes(), nil
}

// VerifySignature check signature if DoNotEncrypt=false
func VerifySignature(signature []byte, pub *rsa.PublicKey, hashed [32]byte) error {
	if DoNotEncrypt {
		return nil
	}
	err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], signature)
	return err
}
