package jwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Token struct {
	JWT            string
	Header         map[string]interface{}
	Payload        map[string]interface{}
	ExpirationDate time.Time
}

func ParseToken(token string) (t *Token, err error) {
	t = &Token{JWT: token}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("jwt malformed: missing encoded parts")
	}

	t.Header = make(map[string]interface{})
	err = parsePart("header", parts[0], &t.Header)
	if err != nil {
		return nil, err
	}

	t.Payload = make(map[string]interface{})
	err = parsePart("payload", parts[1], &t.Payload)
	if err != nil {
		return nil, err
	}

	t.ExpirationDate, err = parseExpirationDate(t.Payload)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// parsePart decodes a string from base64 and parses a JSON.
//
// base64.RawURLEncoding is here because JWT uses 'base64url-encoded'
// with padding characters omitted. See RFC7519, RFC7515.
func parsePart(name string, data string, obj interface{}) error {
	bytes, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("jwt malformed: %s base64: %v", name, err)
	}

	err = json.Unmarshal(bytes, obj)
	if err != nil {
		return fmt.Errorf("jwt malformed: %s json: %v", name, err)
	}

	return nil
}

func parseExpirationDate(payload map[string]interface{}) (time.Time, error) {
	expRaw, ok := payload["exp"]
	if !ok {
		return time.Time{}, fmt.Errorf("jwt malformed: payload.exp is required")
	}

	var unixSec int64
	var err error
	switch v := expRaw.(type) {
	case float64:
		unixSec = int64(v)
	case string:
		unixSec, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("jwt malformed: payload.exp should be unix timestamp")
		}
	}

	return time.Unix(unixSec, 0), nil
}
