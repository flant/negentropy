package jwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type Token struct {
	JWT            string
	Header         map[string]interface{}
	Payload        map[string]interface{}
	ThirdPart      []byte
	ExpirationDate time.Time
	IssuedAtDate   time.Time
	StartRefreshAt time.Time
}

func ParseToken(token string) (t *Token, err error) {
	t = &Token{JWT: token}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("jwt malformed: missing encoded parts")
	}

	t.Header = make(map[string]interface{})
	err = parseJSONPart("header", parts[0], &t.Header)
	if err != nil {
		return nil, err
	}

	t.Payload = make(map[string]interface{})
	err = parseJSONPart("payload", parts[1], &t.Payload)
	if err != nil {
		return nil, err
	}

	t.ThirdPart, err = parseBase64Part("thirdpart", parts[2])
	if err != nil {
		return nil, err
	}

	t.ExpirationDate, err = parseExpirationDate(t.Payload)
	if err != nil {
		return nil, err
	}

	t.IssuedAtDate, err = parseIssuedAtDate(t.Payload)
	if err != nil {
		return nil, err
	}

	t.StartRefreshAt = t.calculateRefreshTime()

	logrus.Debugf("parse token: iat:%s exp:%s refresh:%s now:%s",
		t.IssuedAtDate.Format(time.RFC3339),
		t.ExpirationDate.Format(time.RFC3339),
		t.StartRefreshAt.Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
	)

	return t, nil
}

// parsePart decodes a string from base64 and parses a JSON.
//
// base64.RawURLEncoding is here because JWT uses 'base64url-encoded'
// with padding characters omitted. See RFC7519, RFC7515.
func parseJSONPart(name string, data string, obj interface{}) error {
	bytes, err := parseBase64Part(name, data)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, obj)
	if err != nil {
		return fmt.Errorf("jwt malformed: %s json: %v", name, err)
	}

	return nil
}

func parseBase64Part(name string, data string) ([]byte, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("jwt malformed: %s base64: %v", name, err)
	}
	return bytes, nil
}

func parseExpirationDate(payload map[string]interface{}) (time.Time, error) {
	expRaw, ok := payload["exp"]
	if !ok {
		return time.Time{}, fmt.Errorf("jwt malformed: payload.exp is required")
	}

	t, err := parseUnixTime(expRaw)
	if err != nil {
		return t, fmt.Errorf("jwt payload.exp is malformed: %v", err)
	}
	return t, nil
}

func parseIssuedAtDate(payload map[string]interface{}) (time.Time, error) {
	iatRaw, ok := payload["iat"]
	if !ok {
		return time.Time{}, fmt.Errorf("jwt malformed: payload.iat is required")
	}

	t, err := parseUnixTime(iatRaw)
	if err != nil {
		return t, fmt.Errorf("jwt payload.iat is malformed: %v", err)
	}
	return t, nil
}

func parseUnixTime(in interface{}) (time.Time, error) {
	var unixSec int64
	var err error
	switch v := in.(type) {
	case float64:
		unixSec = int64(v)
	case string:
		unixSec, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("should be unix timestamp")
		}
	}

	return time.Unix(unixSec, 0), nil
}

// Calculate random time in the third quarter of delta between issued and expired.
//     issued      refresh expired
//     |           |       |
//  ---*----.----.-x--.----*---->
func (t *Token) calculateRefreshTime() time.Time {
	issued := t.IssuedAtDate.Unix()
	expired := t.ExpirationDate.Unix()
	delta := expired - issued
	if issued == 0 || expired == 0 || delta <= 0 {
		return time.Now()
	}

	// New random generator with seed from bytes of the ThirdPart of JWT.
	r := rand.New(rand.NewSource(int64FromBytes(t.ThirdPart)))
	refreshUnix := issued + delta/2 + r.Int63n(delta/4)
	return time.Unix(refreshUnix, 0)
}

const int64Bytes = 8

// int64FromBytes fills int64 with bytes until len is exceeded or int64 is filled.
func int64FromBytes(in []byte) int64 {
	var x int64
	var s int
	var l = len(in)
	for i := 0; i < int64Bytes; i++ {
		x |= int64(in[i]) << s
		s += 8
		if i+1 == l {
			break
		}
	}
	return x
}
