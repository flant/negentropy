package server

import (
	"fmt"
	"github.com/flant/negentropy/authd/pkg/jwt"
	"github.com/go-chi/chi/v5"
	"net/http"

	api "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/authd/pkg/config"
	"github.com/flant/negentropy/authd/pkg/vault"
)

const LoginURI = "/v1/login/{serverType:[a-z]+}"

func SetupLoginHandler(router chi.Router, authdConfig *config.AuthdConfig, socketConfig *config.AuthdSocketConfig) {
	router.Method("POST", LoginURI, NewLoginHandler(authdConfig, socketConfig))
}

type LoginHandler struct {
	AuthdConfig       *config.AuthdConfig
	AuthdSocketConfig *config.AuthdSocketConfig
}

func NewLoginHandler(authdConfig *config.AuthdConfig, authdSocketConfig *config.AuthdSocketConfig) *LoginHandler {
	return &LoginHandler{
		AuthdConfig:       authdConfig,
		AuthdSocketConfig: authdSocketConfig,
	}
}

// ServeHTTP does some magic behind the /v1/login.
func (l *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var loginRequest = new(api.LoginRequest)
	ServeJSON(w, r, loginRequest, func() (interface{}, int, error) {
		loginRequest.ServerType = chi.URLParam(r, "serverType")
		return l.HandleLogin(loginRequest)
	})
}

// HandleLogin do some requests to Vault and returns one of LoginResponse.
func (l *LoginHandler) HandleLogin(request *api.LoginRequest) (interface{}, int, error) {
	var err error

	fmt.Printf("Request login for '%s' '%s'", request.ServerType, request.Server)

	vaultServer, err := config.DetectServerAddr(l.AuthdConfig.GetServers(), request.ServerType, request.Server)
	if err != nil {
		return nil, 0, err
	}

	vaultClient := &vault.Client{Server: vaultServer}

	token, err := jwt.DefaultStorage.GetJWT()
	if err != nil {
		return nil, 0, NewHTTPError(err, http.StatusForbidden, []string{err.Error()})
	}

	var response interface{}

	if request.Type == api.LoginRequestDefault || request.Type == api.LoginRequestSpecific {
		secret, err := vaultClient.LoginWithJWT(token)
		if err != nil {
			return nil, 0, err
		}
		if secret.Data == nil {
			secret.Data = map[string]interface{}{
				"server": vaultServer,
			}
		}
		response = secret
	}
	if request.Type == api.LoginRequestPending {
		response, err = vaultClient.CheckPendingLogin(token)
	}

	if err != nil {
		return nil, 0, err
	}

	//var status int
	//switch response.(type) {
	//case *api.LoginResponseSession:
	//	status = http.StatusOK
	//case *api.LoginResponseMsg:
	//	status = http.StatusForbidden
	//case *api.LoginResponsePending:
	//	status = http.StatusUnauthorized
	//default:
	//	return nil, 0, fmt.Errorf("unrecognized request")
	//}

	return response, http.StatusOK, nil
	//	// do the job
	//
	//// calculate http status
	//
	//// return response, status, err
	//
	//switch request.Type {
	//case api.LoginRequestDefault:
	//	srv := l.AuthdConfig.GetDefaultServer()
	//	request.Server = srv.Domain
	//	return l.HandleOpenSession(request)
	//	return &api.LoginResponseSession{
	//		Server: request.Server,
	//		Token:  "qweqwe-123qwe-123qwe1-1",
	//	}, http.StatusOK, nil
	//case api.LoginRequestSpecific:
	//	err := l.CheckServerContraint(request.Server)
	//	return &api.LoginResponseMsg{
	//		Messages: []string{"Denied"},
	//	}, http.StatusForbidden, nil
	//case api.LoginRequestPending:
	//
	//	err := l.CheckServerContraint(request.Server)
	//	return &api.LoginResponsePending{
	//		Server:           request.Server,
	//		PendingLoginUuid: "pok-123pok-123",
	//		Mfa:              []api.Mfa{
	//			{
	//				Type:      "web",
	//				Uuid:      "123-wer-123-wer-tttgrt",
	//				Completed: false,
	//			},
	//		},
	//		Approvals:        []api.Approval{
	//			{
	//				Type:      "web",
	//				Uuid:      "asdfg-hgfd-qwert",
	//				Message:   "Should be approved",
	//				Required:  0,
	//				Completed: 0,
	//			},
	//		},
	//	}, http.StatusUnauthorized, nil
	//}
	//
	//return nil, 0, fmt.Errorf("unrecognized request")
}
