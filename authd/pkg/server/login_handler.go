package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-multierror"

	api "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/authd/pkg/client_error"
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
		return nil, http.StatusForbidden, client_error.NewHTTPError(err, http.StatusForbidden, []string{err.Error()})
	}

	var response interface{}

	if err := l.checkClaimedRoles(request.Roles); err != nil {
		return nil, http.StatusForbidden, client_error.NewHTTPError(err, http.StatusForbidden, []string{err.Error()})
	}

	if request.Type == api.LoginRequestDefault || request.Type == api.LoginRequestSpecific {
		log.Debugf(ctx)("LoginWithJWT")
		response, err = vaultClient.LoginWithJWTAndClaims(ctx, token, request.Roles)
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

// checkClaimedRoles check is role in allowed list
func (l *LoginHandler) checkClaimedRoles(roles []api.RoleWithClaim) error {
	if len(l.AuthdSocketConfig.GetAllowedRoles()) == 0 {
		return nil
	}
	multiError := multierror.Error{}
	for _, claimedRole := range roles {
		found := false
		for _, allowedRole := range l.AuthdSocketConfig.GetAllowedRoles() {
			if claimedRole.Role == allowedRole.Role {
				found = true
				break
			}
		}
		if !found {
			multiError.Errors = append(multiError.Errors, fmt.Errorf("role %s is not in allowed list", claimedRole.Role))
		}
	}
	return multiError.ErrorOrNil()
}
