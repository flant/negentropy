package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/caos/oidc/pkg/crypto"
	"github.com/caos/oidc/pkg/op"
	"github.com/gorilla/mux"
	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/e2e/tests/lib/oidc_mock/mock"
)

func main() {
	os.Setenv("CAOS_OIDC_DEV", "1")
	ctx := context.Background()
	port := "9998"
	config := &op.Config{
		Issuer:    "http://localhost:9998/",
		CryptoKey: sha256.Sum256([]byte("test")),
	}
	storage := mock.NewAuthStorage()
	handler, err := op.NewOpenIDProvider(ctx, config, storage,
		op.WithCustomTokenEndpoint(op.NewEndpoint("test")),
	)
	if err != nil {
		log.Fatal(err)
	}

	router := handler.HttpHandler().(*mux.Router)
	router.Methods("GET").Path("/login").HandlerFunc(HandleLogin)
	router.Methods("POST").Path("/login").HandlerFunc(HandleCallback)
	router.Methods("GET").Path("/custom_id_token").HandlerFunc(CustomIdTokenCreater(handler.Signer().Signer()))
	router.Methods("GET").Path("/custom_access_token").HandlerFunc(CustomAccessTokenCreater(storage, handler))
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
	<-ctx.Done()
}

func CustomIdTokenCreater(signer jose.Signer) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()

		oktaTokenSample := map[string]interface{}{
			"sub":   "00urtzy5ez5mlIsMu0x7",
			"name":  "Axxxx Sxxxxxxxx",
			"email": "alxxxx.sxxxxxxxx@flant.com",
			"ver":   1,
			"iss":   "https://login.flant.com",
			"aud":   "0xaraxvxbgx0Ewem90x8",
			"iat":   1638950310,
			"exp":   2638453910,
			"jti":   "ID.qa-O6OObmW-vbP15asILDYAvowZdp5D7qQZWId4vhqk",
			"amr": []string{
				"mfa",
				"swk",
				"pwd",
			},
			"idp":                "00o3kb39skMQ3yUAq0x7",
			"nonce":              "96MJJV7HuWIXe1UtuMOUNxCjPWaQ7w3cjClfDrbKKfk=",
			"preferred_username": "axxxx.sxxxxxxxx@flant.com",
			"given_name":         "Axxxx",
			"family_name":        "Sxxxxxxxx",
			"updated_at":         1623814694,
			"email_verified":     true,
			"auth_time":          1638450298,
			"roles":              "[\"user\"]",
			"uuid":               "1c3d88ff-2a3a-4061-b2b5-fc8adeb83e23",
		}
		oktaTokenSample["auth_time"] = time.Now().Unix()

		for k, v := range values {
			if len(v) == 1 {
				oktaTokenSample[k] = v[0]
			} else {
				oktaTokenSample[k] = v
			}
		}
		payload, err := json.Marshal(oktaTokenSample)
		if err != nil {
			panic(err)
		}

		idToken, err := crypto.SignPayload(payload, signer)
		if err != nil {
			panic(err)
		}
		w.Write([]byte(idToken))
	}
}

func CustomAccessTokenCreater(storage *mock.AuthStorage, provider op.OpenIDProvider) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		subject := "subject_" + time.Now().String()

		storage.UserExtraData[subject] = map[string]interface{}{}
		for k, v := range values {
			if len(v) == 1 {
				storage.UserExtraData[subject][k] = v[0]
			} else {
				storage.UserExtraData[subject][k] = v
			}
		}

		createAccessToken := true
		authorizer := provider
		authReq := &mock.AuthRequest{
			Subject:  subject,
			ClientID: "aud666",
		}

		client, err := storage.GetClientByClientID(r.Context(), authReq.GetClientID())

		resp, err := op.CreateTokenResponse(r.Context(), authReq, client, authorizer, createAccessToken, "", "")
		if err != nil {
			panic(err)
		}

		w.Write([]byte(resp.AccessToken))
	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	tpl := `
	<!DOCTYPE html>
	<html>
		<head>
			<meta charset="UTF-8">
			<title>Login</title>
		</head>
		<body>
			<form method="POST" action="/login">
				<input name="client"/>
				<button type="submit">Login</button>
			</form>
		</body>
	</html>`
	t, err := template.New("login").Parse(tpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HandleCallback(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	client := r.FormValue("client")
	http.Redirect(w, r, "/authorize/callback?id="+client, http.StatusFound)
}
