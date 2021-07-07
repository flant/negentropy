package v1

import (
	"encoding/json"
	"fmt"
)

/**

через дефолтный сервер
{
  policies:
  - policy: iam.view
    claim: {...}
}

через конкретный сервер
{
  server: auth.negentropy.flant.com
  policies:
  - policy: iam.view
    claim: {...}
}

продолжение инициированного логина
{
  server: auth.negentropy.flant.com
  pendingLoginUuid: dd8d95a5-db39-4543-846c-b564ee52293d
}

*/
type LoginRequest struct {
	Server           string   `json:"server,omitempty"`
	Policies         []Policy `json:"policies,omitempty"`
	PendingLoginUuid string   `json:"pendingLoginUuid,omitempty"`
	Type             string   `json:"-"`
	ServerType       string   `json:"-"`
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

	if polVal, ok := m["policies"]; ok {
		// Transform
		policiesBytes, _ := json.Marshal(polVal)
		req.Policies = make([]Policy, 0)
		err := json.Unmarshal(policiesBytes, &req.Policies)
		if err != nil {
			return fmt.Errorf("parse policies: %v", err)
		}

		if len(req.Policies) == 0 {
			return fmt.Errorf("policies are required")
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

type Policy struct {
	Policy string
	Claim  map[string]string
}

// Client helpers
func NewLoginRequest() *LoginRequest {
	return &LoginRequest{}
}

func (l *LoginRequest) WithServer(server string) *LoginRequest {
	l.Server = server
	return l
}

func (l *LoginRequest) WithPolicies(policies ...Policy) *LoginRequest {
	l.Policies = policies
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

func NewPolicy(policy string, claim map[string]string) Policy {
	return Policy{
		Policy: policy,
		Claim:  claim,
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
