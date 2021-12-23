package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/authd/pkg/config"
	"github.com/flant/negentropy/authd/pkg/jwt"
	"github.com/flant/negentropy/authd/pkg/log"
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
	loginRequest := new(api.LoginRequest)
	ServeJSON(w, r, loginRequest, func(ctx context.Context) (interface{}, int, error) {
		loginRequest.ServerType = chi.URLParam(r, "serverType")
		return l.HandleLogin(ctx, loginRequest)
	})
}

// HandleLogin do some requests to Vault and returns one of LoginResponse.
func (l *LoginHandler) HandleLogin(ctx context.Context, request *api.LoginRequest) (interface{}, int, error) {
	var err error

	log.Debugf(ctx)("Request 'login' for '%s' via '%s'", request.ServerType, request.Server)

	vaultServer, err := config.DetectServerAddr(l.AuthdConfig.GetServers(), request.ServerType, request.Server)
	if err != nil {
		return nil, 0, err
	}

	log.Debugf(ctx)("Use Vault server '%s'", vaultServer)

	vaultClient := vault.NewClient(vaultServer)

	token, err := jwt.DefaultStorage.GetJWT()
	if err != nil {
		return nil, 0, NewHTTPError(err, http.StatusForbidden, []string{err.Error()})
	}

	var response interface{}

	claimedRoles := l.claimedRoles(request)

	if request.Type == api.LoginRequestDefault || request.Type == api.LoginRequestSpecific {
		log.Debugf(ctx)("LoginWithJWT")
		response, err = vaultClient.LoginWithJWTAndClaims(ctx, token, claimedRoles)
	}
	if request.Type == api.LoginRequestPending {
		log.Debugf(ctx)("CheckPendingLogin")
		response, err = vaultClient.CheckPendingLogin(token)
	}

	if err != nil {
		return nil, 0, err
	}

	return response, http.StatusOK, nil
}

// claimedRoles returns all roles specified at config, if '*' is requested
func (l *LoginHandler) claimedRoles(request *api.LoginRequest) []api.RoleWithClaim {
	claimedRoles := request.Roles
	if len(claimedRoles) == 1 && claimedRoles[0].Role == "*" {
		claimedRoles = []api.RoleWithClaim{}
		for _, r := range l.AuthdSocketConfig.GetAllowedRoles() {
			claimedRoles = append(claimedRoles, api.RoleWithClaim{
				Role: r.Role,
			})
		}
	}
	return claimedRoles
}
