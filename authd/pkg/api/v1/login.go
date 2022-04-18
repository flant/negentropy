package v1

import (
	"encoding/json"
	"fmt"
)

/**

LoginRequest examples:

Login using the default server.
{
  roles:
  - role: iam.view
    claim: {...}
}

Login using the specific server.
{
  server: auth.negentropy.flant.com
  roles:
  - role: iam.view
    claim: {...}
}

Continue pending login.
{
  server: auth.negentropy.flant.com
  pendingLoginUuid: dd8d95a5-db39-4543-846c-b564ee52293d
}

*/
type LoginRequest struct {
	Server           string          `json:"server,omitempty"`
	Roles            []RoleWithClaim `json:"roles,omitempty"`
	PendingLoginUuid string          `json:"pendingLoginUuid,omitempty"`
	Type             string          `json:"-"`
	ServerType       string          `json:"-"`
}

const (
	LoginRequestDefault  = "DefaultServer"
	LoginRequestSpecific = "SpecificServer"
	LoginRequestPending  = "PengingLogin"
)

func (req *LoginRequest) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	req.Type = LoginRequestDefault
	if srvVal, ok := m["server"]; ok {
		srv, ok := srvVal.(string)
		if !ok {
			return fmt.Errorf("server is not a string")
		}
		req.Server = srv
	}

	if uuidVal, ok := m["pendingLoginUuid"]; ok {
		uuid, ok := uuidVal.(string)
		if !ok {
			return fmt.Errorf("pendingLoginUuid is not a string")
		}
		req.PendingLoginUuid = uuid
		if req.Server == "" {
			return fmt.Errorf("pendingLoginUuid requires a server")
		}
		req.Type = LoginRequestPending
		return nil
	}

	if polVal, ok := m["roles"]; ok {
		// Transform
		rolesBytes, _ := json.Marshal(polVal)
		req.Roles = make([]RoleWithClaim, 0)
		err := json.Unmarshal(rolesBytes, &req.Roles)
		if err != nil {
			return fmt.Errorf("parse policies: %v", err)
		}

		if len(req.Roles) == 0 {
			return fmt.Errorf("roles are required")
		}

		if req.Server == "" {
			req.Type = LoginRequestDefault
		} else {
			req.Type = LoginRequestSpecific
		}
		return nil
	}

	return fmt.Errorf("malformed login request")
}

type RoleWithClaim struct {
	Role        string                 `json:"role"`
	TenantUUID  string                 `json:"tenant_uuid"`
	ProjectUUID string                 `json:"project_uuid"`
	Claim       map[string]interface{} `json:"claim"`
}

// Client helpers
func NewLoginRequest() *LoginRequest {
	return &LoginRequest{}
}

func (l *LoginRequest) WithServer(server string) *LoginRequest {
	l.Server = server
	return l
}

func (l *LoginRequest) WithRoles(roles ...RoleWithClaim) *LoginRequest {
	l.Roles = roles
	return l
}

func (l *LoginRequest) WithServerType(serverType string) *LoginRequest {
	l.ServerType = serverType
	return l
}

func (l *LoginRequest) WithPendingLoginUuid(loginUuid string) *LoginRequest {
	l.PendingLoginUuid = loginUuid
	return l
}

func NewRoleWithClaim(role string, claim map[string]interface{}) RoleWithClaim {
	return RoleWithClaim{
		Role:  role,
		Claim: claim,
	}
}

/*
{
	server: ew1a1.auth.negentropy.flant.com
  	token: 123e4567-e89b-12d3-a456-426614174000
}


{
	messages: [“Net. Vam syida nelzia!”]
}

{
  server: ew1a1.auth.negentropy.flant.com
  pendingLoginUuid: dd8d95a5-db39-4543-846c-b564ee52293d
  mfa:
  - type: web
    uuid: 2c6a1937-1dae-4fff-a231-e59cede734c9
    completed: false
  approvals:
  - type: web
    uuid: 5c0d0d7b-3789-44cd-b3d9-dcb96d166ff0
    message: “Требуется апрув тимлидов”
    required: 3
    completed: 0
  - type: web
    uuid: 7dc0ad81-002a-41cf-b10d-da1a42cf8c40
    message: “Тратата”
    required: 1
    completed: 0
  - type: web
    uuid: 4f899cf1-c3e6-408e-886d-c96bd02a8a11
    message: “Тратата2”
    required: 1
    completed: 0
}
*/
type LoginResponseSession struct {
	Server string `json:"server,omitempty"`
	Token  string `json:"token,omitempty"`
}

type LoginResponseMsg struct {
	Messages []string `json:"messages,omitempty"`
}

type LoginResponsePending struct {
	Server           string     `json:"server,omitempty"`
	PendingLoginUuid string     `json:"pendingLoginUuid,omitempty"`
	Mfa              []Mfa      `json:"mfa,omitempty"`
	Approvals        []Approval `json:"approvals,omitempty"`
}

type Mfa struct {
	Type      string `json:"type"`
	Uuid      string `json:"uuid"`
	Completed bool   `json:"completed"`
}

type Approval struct {
	Type      string `json:"type"`
	Uuid      string `json:"uuid"`
	Message   string `json:"message"`
	Required  int    `json:"required"`
	Completed int    `json:"completed"`
}

// Client helpers

func UnmarshalLoginResponse(data []byte) (interface{}, error) {
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}

	if _, ok := m["messages"]; ok {
		var obj *LoginResponseMsg
		err := json.Unmarshal(data, obj)
		return obj, err
	}

	if _, ok := m["server"]; ok {
		if _, ok := m["token"]; ok {
			var obj *LoginResponseSession
			err := json.Unmarshal(data, obj)
			return obj, err
		}
		if _, ok := m["PendingLoginUuid"]; ok {
			var obj *LoginResponsePending
			err := json.Unmarshal(data, obj)
			return obj, err
		}
	}

	return nil, fmt.Errorf("malformed login response")
}
