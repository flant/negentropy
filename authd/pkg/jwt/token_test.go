package jwt

import (
	"fmt"
	"testing"
)

/*
{
  "sub": "1234567890",
  "name": "John Doe",
  "admin": true,
  "exp": 1516239022
}
*/
var testToken = `eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImV4cCI6MTUxNjIzOTAyMn0.WZTc_d2U578iTLUeoRvCUbetepYaF8MgNm7aiHT0FvvL-FWRCfiImqz9ybTNxATvBH8UQM_vZvIXHAVaRrDacw`

/*
{
  "sub": "1234567890",
  "name": "John Doe",
  "admin": true,
  "iat": 1516239022
}
*/
var testTokenNoExp = `eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.Hx1u1lvv3e7wCmZZ1e7AncjtIgeRTChMBnbBboJagbkvQNxlNHoHrRKOf22aOQAtDPCqyPpPhCUY0DeJsJZePw`

func Test_Parse_token(t *testing.T) {
	tok, err := ParseToken(testToken)
	if err != nil {
		t.Fatalf("testToken should be parsed: %v", err)
	}

	fmt.Printf("exp is '%s'\n", tok.ExpirationDate.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
}
