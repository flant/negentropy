package jwt

import (
	"testing"
)

/*
{
  "iss": "https://auth.negentropy.flant.com/",
  "exp": 9406476572,
  "iat": 1640196941,
  "sub": "92f91630-0b2c-403b-896d-b1f93c0ba0e5",
  "aud": "",
  "jti": "a51aa9b84ea9ca5bb4bc8f5a2a547d1a5ffdc426d2195a3129fa634f63af07c1"
}
*/
var testToken = `eyJhbGciOiJFZERTQSIsImtpZCI6IjBlNzI1YmQ4MWVkZjQwN2JhZjYyMDYzOGFjZWQ0NTM5In0.eyJpc3MiOiJodHRwczovL2F1dGgubmVnZW50cm9weS5mbGFudC5jb20vIiwiZXhwIjo5NDA2NDc2NTcyLCJpYXQiOjE2NDAxOTY5NDEsInN1YiI6IjkyZjkxNjMwLTBiMmMtNDAzYi04OTZkLWIxZjkzYzBiYTBlNSIsImF1ZCI6IiIsImp0aSI6ImE1MWFhOWI4NGVhOWNhNWJiNGJjOGY1YTJhNTQ3ZDFhNWZmZGM0MjZkMjE5NWEzMTI5ZmE2MzRmNjNhZjA3YzEifQ.BPJV6SglPLuvUZ0wOqk2pfn7ItQFO2goH7Wsxs9j8zFAIp1U_p9kZavIVVEow0XmHhBUPKKxplaBH_Vdbe7aBQ`

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
	_, err := ParseToken(testToken)
	if err != nil {
		t.Fatalf("testToken should be parsed: %v", err)
	}
	_, err = ParseToken(testTokenNoExp)
	if err == nil || err.Error() != "jwt malformed: payload.exp is required" {
		t.Fatalf("err should be: jwt malformed: payload.exp is required, actual: %v", err)
	}
}
