package ext_server_access

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/usecase"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

// TODO: changed group identifier once project indentifier changes in the flant_iam

type serverBackend struct {
	logical.Backend
	storage       *io.MemoryStore
	jwtController *jwt.Controller
}

func ServerPaths(b logical.Backend, storage *io.MemoryStore, jwtController *jwt.Controller) []*framework.Path {
	bb := &serverBackend{
		Backend:       b,
		storage:       storage,
		jwtController: jwtController,
	}

	return bb.paths()
}

func (b *serverBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: path.Join("tenant", uuid.Pattern("tenant_uuid"),
				"project", uuid.Pattern("project_uuid"), "register_server"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"labels": {
					Type:        framework.TypeMap,
					Description: "Map of labels",
					Required:    false,
				},
				"annotations": {
					Type:        framework.TypeMap,
					Description: "Map of annotations",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleRegister(),
					Summary:  "Register server",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleRegister(),
					Summary:  "Register server",
				},
			},
		},
		{
			Pattern: path.Join(
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"),
				"server", uuid.Pattern("server_uuid")),
			Fields: map[string]*framework.FieldSchema{
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
				},
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
				"server_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a server",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
				},
				"labels": {
					Type:        framework.TypeMap,
					Description: "Map of labels",
				},
				"annotations": {
					Type:        framework.TypeMap,
					Description: "Map of annotations",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Read a server",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update a server",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Delete a server",
				},
			},
		},
		{
			Pattern: path.Join(
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "servers/?"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "List servers",
				},
			},
		},
		{
			Pattern: path.Join(
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"),
				"server", uuid.Pattern("server_uuid"), "fingerprint",
			),
			Fields: map[string]*framework.FieldSchema{
				"server_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a server",
					Required:    true,
				},
				"fingerprint": {
					Type:        framework.TypeNameString,
					Description: "Fingerprint of a server",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleFingerprintRead(),
					Summary:  "Read a server's fingerprint",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleFingerprintUpdate(),
					Summary:  "Update a server's fingerprint",
				},
			},
		},
		{
			Pattern: path.Join(
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"),
				"server", uuid.Pattern("server_uuid"), "connection_info",
			),
			Fields: map[string]*framework.FieldSchema{
				"server_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a server",
					Required:    true,
				},
				"hostname": {
					Type:        framework.TypeString,
					Description: "IP or a hostname",
					Required:    true,
				},
				"port": {
					Type:        framework.TypeString,
					Description: "Port, optional, 22 by default",
				},
				"jump_hostname": {
					Type:        framework.TypeString,
					Description: "IP or a hostname, optional",
				},
				"jump_port": {
					Type:        framework.TypeString,
					Description: "Port, optional, 22 by default if jump_hostname is defined",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleConnectionInfoUpdate(),
					Summary:  "Update a server's connection_info",
				},
			},
		},
	}
}

func (b *serverBackend) handleRegister() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("handleRegister started")
		defer b.Logger().Debug("handleRegister exit")
		if !liveConfig.isConfigured() {
			err := errors.New("backend not yet configured")
			return logical.ErrorResponse(err.Error()), err
		}
		config, err := liveConfig.GetServerAccessConfig(ctx, req.Storage)
		if err != nil {
			return logical.ErrorResponse(err.Error()), err
		}

		var (
			labels      = make(map[string]string)
			annotations = make(map[string]string)
		)
		for k, v := range data.Get("labels").(map[string]interface{}) {
			labels[k] = v.(string)
		}
		for k, v := range data.Get("annotations").(map[string]interface{}) {
			annotations[k] = v.(string)
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		service := usecase.NewServerService(tx)

		issueFn := jwt.CreateIssueMultipassFunc(b.jwtController, tx)

		serverUUID, jwtToken, err := service.Create(issueFn, data.Get("tenant_uuid").(string), data.Get("project_uuid").(string),
			data.Get("identifier").(string), labels, annotations, config.RolesForServers)
		if err != nil {
			err = fmt.Errorf("cannot register server:%w", err)
			b.Logger().Error(err.Error())
			return backentutils.ResponseErr(req, err)
		}

		err = tx.Commit()
		if err != nil {
			msg := fmt.Sprintf("cannot commit transaction:%s", err.Error())
			b.Logger().Error(msg)
			return logical.ErrorResponse(msg), err
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"multipassJWT": jwtToken,
			"uuid":         serverUUID,
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *serverBackend) handleFingerprintRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("handleFingerprintRead started")
		defer b.Logger().Debug("handleFingerprintRead exit")
		tx := b.storage.Txn(false)
		defer tx.Abort()
		repo := repo.NewServerRepository(tx)

		server, err := repo.GetByUUID(data.Get("server_uuid").(string))
		if err != nil {
			err := fmt.Errorf("cannot get server from db: %s", err)
			b.Logger().Error("err", err.Error())
			return backentutils.ResponseErr(req, err)
		}

		err = io.CommitWithLog(tx, b.Logger())
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"fingerprint": server.Fingerprint}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleFingerprintUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("handleFingerprintUpdate started")
		defer b.Logger().Debug("handleFingerprintUpdate exit")
		fingerprintRaw, ok := data.GetOk("fingerprint")
		if !ok {
			err := errors.New("fingerprint is not provided or is invalid")
			b.Logger().Error("err", err.Error())
			return backentutils.ResponseErr(req, err)
		}

		fingerprint := fingerprintRaw.(string)

		tx := b.storage.Txn(false)
		defer tx.Abort()
		repo := repo.NewServerRepository(tx)

		server, err := repo.GetByUUID(data.Get("server_uuid").(string))
		if err != nil {
			err := fmt.Errorf("cannot get server from db: %s", err)
			b.Logger().Error("err", err.Error())
			return backentutils.ResponseErr(req, err)
		}
		if server.Archived() {
			return backentutils.ResponseErr(req, consts.ErrIsArchived)
		}
		server.Fingerprint = fingerprint

		err = repo.Update(server)
		if err != nil {
			err := fmt.Errorf("cannot update server: %s", err)
			b.Logger().Error("err", err.Error())
			return backentutils.ResponseErr(req, err)
		}

		err = io.CommitWithLog(tx, b.Logger())
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"fingerprint": server.Fingerprint}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("handleRead started")
		defer b.Logger().Debug("handleRead exit")
		tx := b.storage.Txn(false)
		defer tx.Abort()
		repo := repo.NewServerRepository(tx)

		server, err := repo.GetByUUID(data.Get("server_uuid").(string))
		if err != nil {
			err := fmt.Errorf("cannot get server from db: %s", err)
			b.Logger().Error("err", err.Error())
			return backentutils.ResponseErr(req, err)
		}

		err = io.CommitWithLog(tx, b.Logger())
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"server": server}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("handleUpdate started")
		defer b.Logger().Debug("handleUpdate exit")

		var (
			labels      = make(map[string]string)
			annotations = make(map[string]string)
		)
		for k, v := range data.Get("labels").(map[string]interface{}) {
			labels[k] = v.(string)
		}
		for k, v := range data.Get("annotations").(map[string]interface{}) {
			annotations[k] = v.(string)
		}

		server := &model.Server{
			UUID:        data.Get("server_uuid").(string),
			TenantUUID:  data.Get(iam_repo.TenantForeignPK).(string),
			ProjectUUID: data.Get(iam_repo.ProjectForeignPK).(string),
			Version:     data.Get("resource_version").(string),
			Identifier:  data.Get("identifier").(string),
			Labels:      labels,
			Annotations: annotations,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.NewServerService(tx).Update(server)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Error(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		resp := &logical.Response{Data: map[string]interface{}{"server": server}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("handleDelete started")
		defer b.Logger().Debug("handleDelete exit")
		id := data.Get("server_uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		service := usecase.NewServerService(tx)

		err := service.Delete(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Error(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *serverBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("handleList started")
		defer b.Logger().Debug("handleList exit")
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)
		projectID := data.Get(iam_repo.ProjectForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := repo.NewServerRepository(tx)

		list, err := repo.List(tenantID, projectID, false)
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"servers": list,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleConnectionInfoUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("handleConnectionInfoUpdate started")
		defer b.Logger().Debug("handleConnectionInfoUpdate exit")
		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := repo.NewServerRepository(tx)
		serverUUID := data.Get("server_uuid").(string)
		server, err := repo.GetByUUID(serverUUID)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		connectionInfo := model.ConnectionInfo{
			Hostname:     data.Get("hostname").(string),
			Port:         data.Get("port").(string),
			JumpHostname: data.Get("jump_hostname").(string),
			JumpPort:     data.Get("jump_port").(string),
		}

		connectionInfo.FillDefaultPorts()
		server.ConnectionInfo = connectionInfo
		err = repo.Update(server)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Error(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		resp := &logical.Response{Data: map[string]interface{}{"server": server}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
