package extension_server_access

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"k8s.io/apimachinery/pkg/labels"

	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
)

const (
	serverAccessSSHRoleKey = "iam_auth.extensions.server_access.ssh_role"
)

type serverAccessBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func ServerAccessPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &serverAccessBackend{
		Backend: b,
		storage: storage,
	}

	return bb.paths()
}

func (b *serverAccessBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: path.Join("configure_extension", "server_access"),
			Fields: map[string]*framework.FieldSchema{
				"role_for_ssh_access": {
					Type:        framework.TypeString,
					Description: "Role to use for SSH access",
					Default:     "ssh",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleConfig(),
					Summary:  "Register server extension",
				},
			},
		},

		{
			Pattern: path.Join("tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "server", uuid.Pattern("server_uuid"), "posix_users"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a project",
					Required:    true,
				},
				"server_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a server",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleReadPosixUsers(),
					Summary:  "GET all posix users",
				},
			},
		},

		{
			Pattern: path.Join("tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "server", uuid.Pattern("server_uuid")),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a project",
					Required:    true,
				},
				"server_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a server",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleServerJWT(),
					Summary:  "GET server JWT",
				},
			},
		},

		// query servers
		{
			Pattern: path.Join("tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "query_server/?"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a project",
					Required:    true,
				},
				"names": {
					Type:        framework.TypeCommaStringSlice,
					Description: "server names array",
					Query:       true,
				},
				"labelSelector": {
					Type:        framework.TypeString,
					Query:       true,
					Description: "label selector",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.queryServer(),
					Summary:  "Query servers in project by names or labels",
				},
			},
		},
		{
			Pattern: path.Join("tenant", uuid.Pattern("tenant_uuid"), "query_server/?"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a tenant",
					Required:    true,
				},
				"labelSelector": {
					Type:        framework.TypeString,
					Query:       true,
					Description: "label selector",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.queryServer(),
					Summary:  "Query servers in tenant by labels",
				},
			},
		},
		{
			Pattern: "query_server/?",
			Fields: map[string]*framework.FieldSchema{
				"labelSelector": {
					Type:        framework.TypeString,
					Query:       true,
					Description: "label selector",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.queryServer(),
					Summary:  "Query servers by labels",
				},
			},
		},
	}
}

func (b *serverAccessBackend) handleConfig() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		rawRoleForSSHAccess := data.Get("role_for_ssh_access")

		err := req.Storage.Put(ctx, &logical.StorageEntry{
			Key:      serverAccessSSHRoleKey,
			Value:    []byte(rawRoleForSSHAccess.(string)),
			SealWrap: true,
		})
		if err != nil {
			return nil, err
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusOK)
	}
}

func (b *serverAccessBackend) handleServerJWT() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get("tenant_uuid").(string)
		projectID := data.Get("project_uuid").(string)
		serverID := data.Get("server_uuid").(string)

		txn := b.storage.Txn(false)
		defer txn.Abort()

		repo := model2.NewServerRepository(txn)
		server, err := repo.GetByUUID(serverID)
		if err != nil {
			return nil, err
		}

		if server.TenantUUID != tenantID || server.ProjectUUID != projectID {
			return nil, model.ErrNotFound
		}

		token, err := jwt.NewJwtToken(ctx, req.Storage, server.AsMap(), &jwt.TokenOptions{})
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{"token": token},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

type queryServer struct {
	UUID        string `json:"uuid"`
	Identifier  string `json:"identifier"`
	Version     string `json:"resource_version"`
	ProjectUUID string `json:"project_uuid"`
	TenantUUID  string `json:"tenant_uuid"`
}

func (b *serverAccessBackend) queryServer() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		var tenantID, projectID string
		var serverNames []string

		tenantIDRaw, ok := data.GetOk("tenant_uuid")
		if ok {
			tenantID = tenantIDRaw.(string)
		}
		projectIDRaw, ok := data.GetOk("project_uuid")
		if ok {
			projectID = projectIDRaw.(string)
		}

		serverNamesRaw, ok := data.GetOk("names")
		if ok {
			serverNames = serverNamesRaw.([]string)
		}
		labelSelector := data.Get("labelSelector").(string)

		if projectID == "" {
			// ?names= query param available only for /tenant/<uuid>/project/<uuid>/query_server path
			// ignore it for /tenant/<uuid>/query_server and /query_server
			serverNames = []string{}
		}

		if len(serverNames) > 0 && labelSelector != "" {
			return nil, errors.New("only names or labelSelector must be set")
		}

		b.Logger().Debug("query servers", "names", serverNames, "labelSelector", labelSelector)

		txn := b.storage.Txn(false)
		defer txn.Abort()

		var (
			result   []queryServer
			err      error
			warnings []string
		)

		switch {
		case len(serverNames) > 0:
			result, warnings, err = findSeversByNames(txn, serverNames, tenantID, projectID)

		case labelSelector != "":
			result, err = findServersByLabels(txn, labelSelector, tenantID, projectID)

		default:
			result, err = findServers(txn, tenantID, projectID)
		}
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Warnings: warnings,
			Data:     map[string]interface{}{"servers": result},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverAccessBackend) handleReadPosixUsers() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get("tenant_uuid").(string)
		// projectID := data.Get("project_uuid").(string) // TODO: use it in resolver
		serverID := data.Get("server_uuid").(string)

		txn := b.storage.Txn(false)
		defer txn.Abort()

		sshRole := "ssh"
		en, err := req.Storage.Get(ctx, serverAccessSSHRoleKey)
		if err != nil {
			return logical.ErrorResponse(err.Error()), nil
		}
		if en != nil {
			sshRole = string(en.Value)
		}

		users, serviceAccounts, err := stubResolveUserAndSA(txn, sshRole, tenantID)
		if err != nil {
			return logical.ErrorResponse(err.Error()), nil
		}

		var posixUsers []posixUser
		var warnings []string
		posixBuilder := newPosixUserBuilder(txn, serverID, tenantID)

		for _, user := range users {
			posix, err := posixBuilder.userToPosix(user)
			if err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			posixUsers = append(posixUsers, posix)
		}

		for _, sa := range serviceAccounts {
			posix, err := posixBuilder.serviceAccountToPosix(sa)
			if err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			posixUsers = append(posixUsers, posix)
		}

		resp := &logical.Response{
			Warnings: warnings,
			Data:     map[string]interface{}{"posix_users": posixUsers},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

// TODO: change to real func
func stubResolveUserAndSA(tx *io.MemoryStoreTxn, role, tenantID string) ([]*model.User, []*model.ServiceAccount, error) {
	// for stub, return all users and SA
	userRepo := model.NewUserRepository(tx)
	saRepo := model.NewServiceAccountRepository(tx)

	resUsers := make([]*model.User, 0)
	resSa := make([]*model.ServiceAccount, 0)

	uList, err := userRepo.List(tenantID)
	if err != nil {
		return nil, nil, err
	}

	for _, user := range uList {
		if _, ok := user.Extensions["server_access"]; ok {
			resUsers = append(resUsers, user)
		}
	}

	saList, err := saRepo.List(tenantID)
	if err != nil {
		return nil, nil, err
	}

	for _, sa := range saList {
		if _, ok := sa.Extensions["server_access"]; ok {
			resSa = append(resSa, sa)
		}
	}

	return resUsers, resSa, nil
}

func findServersByLabels(tx *io.MemoryStoreTxn, labelSelector string, tenantID, projectID string) ([]queryServer, error) {
	result := make([]queryServer, 0)

	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return result, err
	}

	repo := model2.NewServerRepository(tx)

	list, err := repo.List(tenantID, projectID)
	if err != nil {
		return result, err
	}

	for _, server := range list {
		set := labels.Set(server.Labels)
		if selector.Matches(set) {
			qs := queryServer{
				UUID:        server.UUID,
				Identifier:  server.Identifier,
				Version:     server.Version,
				ProjectUUID: server.ProjectUUID,
				TenantUUID:  server.TenantUUID,
			}
			result = append(result, qs)
		}
	}

	return result, nil
}

func findServers(tx *io.MemoryStoreTxn, tenantID, projectID string) ([]queryServer, error) {
	repo := model2.NewServerRepository(tx)

	list, err := repo.List(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]queryServer, 0, len(list))

	for _, server := range list {
		qs := queryServer{
			UUID:        server.UUID,
			Identifier:  server.Identifier,
			Version:     server.Version,
			ProjectUUID: server.ProjectUUID,
			TenantUUID:  server.TenantUUID,
		}
		result = append(result, qs)
	}

	return result, nil
}

func findSeversByNames(tx *io.MemoryStoreTxn, names []string, tenantID, projectID string) ([]queryServer, []string, error) {
	result := make([]queryServer, 0)

	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[strings.ToLower(name)] = false
	}
	repo := model2.NewServerRepository(tx)

	list, err := repo.List(tenantID, projectID)
	if err != nil {
		return result, nil, err
	}

	for _, server := range list {
		if _, ok := nameMap[strings.ToLower(server.Identifier)]; ok {
			qs := queryServer{
				UUID:        server.UUID,
				Identifier:  server.Identifier,
				Version:     server.Version,
				ProjectUUID: server.ProjectUUID,
				TenantUUID:  server.TenantUUID,
			}
			result = append(result, qs)
			nameMap[strings.ToLower(server.Identifier)] = true
		}
	}
	var warnings []string
	for name, seen := range nameMap {
		if !seen {
			warnings = append(warnings, fmt.Sprintf("Server: %q not found", name))
		}
	}

	return result, warnings, nil
}
