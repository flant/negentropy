package vault

import (
	"context"
	"strings"
	"testing"
)

func Test_LoginWithJWT(t *testing.T) {

	//token := `eyJhbGciOiJFZERTQSIsImtpZCI6Ijc1ZjQzMDkyLWRlNzAtODM2Mi1kMzIyLTBkMDdjMDlkMTAyZCJ9.eyJhdWQiOiJhdXRoZCIsImV4cCI6MTYxOTI1NzM2NSwiaWF0IjoxNjE5MTcwOTY1LCJpc3MiOiJodHRwczovLzEyNy4wLjAuMTo4MjAwL3YxL2lkZW50aXR5L29pZGMiLCJuYW1lc3BhY2UiOiJyb290Iiwic3ViIjoiOTRkNWQ0M2QtZjViOS1mZWM5LWZhOGMtM2ZmNzExMTQzNzFjIiwidXNlcm5hbWUiOiJlbnRpdHlfODE1Y2JmY2EifQ._rFig3PgSkDCOwil384C6C7Fi97BSYFzUowZ-kiXXIDYahPWVPvHYJBFOhxO8MtTs5-Wyb3_0d_Utl7vfdP5Aw`

	token := `eyJhbGciOiJFZERTQSIsImtpZCI6IjAyNmM1NWI3LWFmZWMtMDIzZi0zYTdlLWUyMDM2MzAyYTZkOSJ9.eyJhdWQiOiJhdXRoZCIsImV4cCI6MTYxOTIxMzI0MSwiaWF0IjoxNjE5MjEzMjExLCJpc3MiOiJodHRwczovLzEyNy4wLjAuMTo4MjAwL3YxL2lkZW50aXR5L29pZGMiLCJuYW1lc3BhY2UiOiJyb290Iiwic3ViIjoiM2JkOTNkMWUtMzIyYi04ZjgxLWNlMjEtNWM4YjI0NjhkYWUxIiwidXNlcm5hbWUiOiJ0ZXN0dXNlciJ9.6tKFiri-cF_Y6SBUCecCprRJ3mSyPbKhWhGVhK8twAtp_0F7PuVhiYom5GURtUri5sLOKl8uYbwcaF3B-0I5Bw`

	vcl := &Client{}

	secret, err := vcl.LoginWithJWT(context.Background(), token)

	if err != nil {
		t.Fatalf("login should be successful. error (type %T): %v", err, err)
	}

	if !strings.HasPrefix(secret.Auth.ClientToken, "s.") {
		t.Fatalf("client token should starts with s., got: '%s'", secret.Auth.ClientToken)
	}
}
