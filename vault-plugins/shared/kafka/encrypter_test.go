package kafka

import (
	"bytes"
	crand "crypto/rand"
	"crypto/rsa"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncrypter(t *testing.T) {
	pk, err := rsa.GenerateKey(crand.Reader, 4096)
	require.NoError(t, err)
	pub := &pk.PublicKey

	tests := []int{300, 445, 446, 447, 700, 891, 892, 893, 13023}

	for _, tt := range tests {
		data := []byte(randString(tt))
		en := NewEncrypter()
		crpyted, chunked, err := en.Encrypt(data, pub)
		require.NoError(t, err)
		if tt <= 446 {
			assert.False(t, chunked)
		} else {
			assert.True(t, chunked)
		}
		decrypted, err := en.Decrypt(crpyted, pk, chunked)
		require.NoError(t, err)
		assert.Equal(t, data, decrypted)
	}
}

func TestSeparator(t *testing.T) {
	sep := strconv.QuoteRuneToASCII('☺')

	buf := bytes.NewBuffer(nil)
	buf.WriteString("aaaaa")
	buf.WriteString(sep)
	buf.WriteString("bbbbb")
	buf.WriteString(sep)
	buf.WriteString("ccccc")
	buf.WriteString(sep)
	buf.WriteString("ddddd")

	spli := bytes.Split(buf.Bytes(), []byte(sep))
	assert.Equal(t, []byte("aaaaa"), spli[0])
	assert.Equal(t, []byte("bbbbb"), spli[1])
	assert.Equal(t, []byte("ccccc"), spli[2])
	assert.Equal(t, []byte("ddddd"), spli[3])
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}
