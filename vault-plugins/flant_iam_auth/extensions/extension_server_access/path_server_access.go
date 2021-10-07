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

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	ext_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	serverAccessSSHRoleKey = "iam_auth.extensions.server_access.ssh_role"
)

type ServerAccessBackend struct {
	logical.Backend
	storage          *io.MemoryStore
	entityIDResolver authn.EntityIDResolver
}

// NewServerAccessBackend returns valid ServerAccessBackend, except entityIDResolver
// need set it before start using  ServerAccessBackend
func NewServerAccessBackend(b logical.Backend, storage *io.MemoryStore) ServerAccessBackend {
	return ServerAccessBackend{
		Backend: b,
		storage: storage,
	}
}

// SetEntityIDResolver set entityIDResolver
func (b *ServerAccessBackend) SetEntityIDResolver(entityIDResolver authn.EntityIDResolver) {
	b.entityIDResolver = entityIDResolver
}

func (b *ServerAccessBackend) Paths() []*framework.Path {
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

func (b *ServerAccessBackend) handleConfig() framework.OperationFunc {
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

// queryServer serve safe and unsafe paths
// safe path is served with tenantID and projectID
func (b *ServerAccessBackend) queryServer() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		var tenantID, projectID string
		var serverNames []string

		// it needs this long way because method is used in several paths
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

		acceptedProjects, err := b.entityIDResolver.AvailableProjectsByEntityID(req.EntityID, txn, req.Storage)
		if err != nil {
			return backentutils.ResponseErrMessage(req, fmt.Sprintf("collect acceptedProjects: %s", err.Error()),
				http.StatusInternalServerError)
		}

		servers, warnings, err := b.allowedServers(txn, tenantID, projectID, serverNames, labelSelector, acceptedProjects)
		if err != nil {
			return backentutils.ResponseErrMessage(req, fmt.Sprintf("filtering servers: %s", err.Error()),
				http.StatusInternalServerError)
		}

		resp := &logical.Response{
			Warnings: warnings,
			Data:     map[string]interface{}{"servers": servers},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *ServerAccessBackend) allowedServers(txn *io.MemoryStoreTxn, tenantID string, projectID string, serverNames []string,
	labelSelector string, allowedProjects map[iam_model.ProjectUUID]struct{}) (interface{}, []string, error) {
	var (
		servers  []*ext_model.Server
		err      error
		warnings []string
	)

	switch {
	case len(serverNames) > 0:
		servers, warnings, err = findSeversByNames(txn, serverNames, tenantID, projectID)
		servers = filterByProjects(servers, allowedProjects)
	case labelSelector != "":
		servers, err = findServersByLabels(txn, labelSelector, tenantID, projectID)
		servers = filterByProjects(servers, allowedProjects)
	default:
		servers, err = findServers(txn, tenantID, projectID)
		servers = filterByProjects(servers, allowedProjects)
	}
	if err != nil {
		return nil, nil, err
	}
	if tenantID != "" && projectID != "" {
		return servers, warnings, nil
	}
	return makeSafeServers(servers), warnings, nil
}

func makeSafeServers(servers []*ext_model.Server) []*model.SafeServer {
	result := make([]*model.SafeServer, 0, len(servers))
	for _, s := range servers {
		result = append(result, &model.SafeServer{
			UUID:        s.UUID,
			Version:     s.Version,
			ProjectUUID: s.ProjectUUID,
			TenantUUID:  s.TenantUUID,
		})
	}
	return result
}

func filterByProjects(servers []*ext_model.Server,
	acceptedProjects map[iam_model.ProjectUUID]struct{}) []*ext_model.Server {
	result := []*ext_model.Server{}
	for i := range servers {
		if _, ok := acceptedProjects[servers[i].ProjectUUID]; ok {
			result = append(result, servers[i])
		}
	}
	return result
}

func (b *ServerAccessBackend) handleReadPosixUsers() framework.OperationFunc {
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
func stubResolveUserAndSA(tx *io.MemoryStoreTxn, role, tenantID string) ([]*iam_model.User, []*iam_model.ServiceAccount, error) {
	// for stub, return all users and SA
	userRepo := iam_repo.NewUserRepository(tx)
	saRepo := iam_repo.NewServiceAccountRepository(tx)

	resUsers := make([]*iam_model.User, 0)
	resSa := make([]*iam_model.ServiceAccount, 0)

	uList, err := userRepo.List(tenantID, false)
	if err != nil {
		return nil, nil, err
	}

	for _, user := range uList {
		if _, ok := user.Extensions["server_access"]; ok {
			resUsers = append(resUsers, user)
		}
	}

	saList, err := saRepo.List(tenantID, false)
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

func findServersByLabels(tx *io.MemoryStoreTxn, labelSelector string, tenantID, projectID string) ([]*ext_model.Server, error) {
	result := make([]*ext_model.Server, 0)

	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return result, err
	}

	repo := ext_repo.NewServerRepository(tx)

	list, err := repo.List(tenantID, projectID)
	if err != nil {
		return result, err
	}

	for _, server := range list {
		set := labels.Set(server.Labels)
		if selector.Matches(set) {
			result = append(result, server)
		}
	}

	return result, nil
}

func findServers(tx *io.MemoryStoreTxn, tenantID, projectID string) ([]*ext_model.Server, error) {
	repo := ext_repo.NewServerRepository(tx)
	list, err := repo.List(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func findSeversByNames(tx *io.MemoryStoreTxn, names []string, tenantID, projectID string) ([]*ext_model.Server, []string, error) {
	result := make([]*ext_model.Server, 0)

	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[strings.ToLower(name)] = false
	}
	repo := ext_repo.NewServerRepository(tx)

	list, err := repo.List(tenantID, projectID)
	if err != nil {
		return result, nil, err
	}

	for _, server := range list {
		if _, ok := nameMap[strings.ToLower(server.Identifier)]; ok {
			result = append(result, server)
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
