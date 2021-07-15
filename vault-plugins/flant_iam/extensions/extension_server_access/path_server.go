package extension_server_access

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// TODO: changed group identifier once project indentifier changes in the flant_iam

type serverBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func ServerPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &serverBackend{
		Backend: b,
		storage: storage,
	}

	return bb.paths()
}

func (b *serverBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: path.Join("tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "register_server"),
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
			},
		},
		{
			Pattern: path.Join(
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "server", uuid.Pattern("server_uuid")),
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
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "servers"),
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
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "List servers",
				},
			},
		},
		{
			Pattern: path.Join(
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "server", uuid.Pattern("server_uuid"), "fingerprint",
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
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "server", uuid.Pattern("server_uuid"), "connection_info",
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
		id := uuid.New()

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

		server := &model2.Server{
			UUID:        id,
			TenantUUID:  data.Get(model.TenantForeignPK).(string),
			ProjectUUID: data.Get(model.ProjectForeignPK).(string),
			Identifier:  data.Get("identifier").(string),
			Labels:      labels,
			Annotations: annotations,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model2.NewServerRepository(tx)

		if err := repo.Create(server, config.RolesForServers); err != nil {
			msg := "cannot create server"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), err
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), err
		}

		// TODO: token creation

		resp := &logical.Response{Data: map[string]interface{}{"server": server}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleFingerprintRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		defer tx.Abort()
		repo := model2.NewServerRepository(tx)

		server, err := repo.GetById(data.Get("server_uuid").(string))
		if err != nil {
			err := fmt.Errorf("cannot get server from db: %s", err)
			b.Logger().Debug("err", err.Error())
			return responseErr(req, err)
		}

		err = commit(tx, b.Logger())
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"fingerprint": server.Fingerprint}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleFingerprintUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		fingerprintRaw, ok := data.GetOk("fingerprint")
		if !ok {
			err := errors.New("fingerprint is not provided or is invalid")
			b.Logger().Debug("err", err.Error())
			return responseErr(req, err)
		}

		fingerprint := fingerprintRaw.(string)

		tx := b.storage.Txn(false)
		defer tx.Abort()
		repo := model2.NewServerRepository(tx)

		server, err := repo.GetById(data.Get("server_uuid").(string))
		if err != nil {
			err := fmt.Errorf("cannot get server from db: %s", err)
			b.Logger().Debug("err", err.Error())
			return responseErr(req, err)
		}

		server.Fingerprint = fingerprint

		err = repo.Update(server)
		if err != nil {
			err := fmt.Errorf("cannot update server: %s", err)
			b.Logger().Debug("err", err.Error())
			return responseErr(req, err)
		}

		err = commit(tx, b.Logger())
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"fingerprint": server.Fingerprint}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		defer tx.Abort()
		repo := model2.NewServerRepository(tx)

		server, err := repo.GetById(data.Get("server_uuid").(string))
		if err != nil {
			err := fmt.Errorf("cannot get server from db: %s", err)
			b.Logger().Debug("err", err.Error())
			return responseErr(req, err)
		}

		err = commit(tx, b.Logger())
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"server": server}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model2.NewServerRepository(tx)

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

		server := &model2.Server{
			UUID:        data.Get("server_uuid").(string),
			TenantUUID:  data.Get(model.TenantForeignPK).(string),
			ProjectUUID: data.Get(model.ProjectForeignPK).(string),
			Version:     data.Get("resource_version").(string),
			Identifier:  data.Get("identifier").(string),
			Labels:      labels,
			Annotations: annotations,
		}

		err := repo.Update(server)
		if err != nil {
			return responseErr(req, err)
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		resp := &logical.Response{Data: map[string]interface{}{"server": server}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serverBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("server_uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model2.NewServerRepository(tx)

		err := repo.Delete(id)
		if err != nil {
			return responseErr(req, err)
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *serverBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)
		projectID := data.Get(model.ProjectForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := model2.NewServerRepository(tx)

		list, err := repo.List(tenantID, projectID)
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuids": list,
			},
		}

		return resp, nil
	}
}

func (b *serverBackend) handleConnectionInfoUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model2.NewServerRepository(tx)
		serverUUID := data.Get("server_uuid").(string)
		server, err := repo.GetById(serverUUID)
		if err != nil {
			return responseErr(req, err)
		}
		connectionInfo := model2.ConnectionInfo{
			Hostname:     data.Get("hostname").(string),
			Port:         data.Get("port").(string),
			JumpHostname: data.Get("jump_hostname").(string),
			JumpPort:     data.Get("jump_port").(string),
		}

		connectionInfo.FillDefaultPorts()
		server.ConnectionInfo = connectionInfo
		err = repo.Update(server)
		if err != nil {
			return responseErr(req, err)
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		resp := &logical.Response{Data: map[string]interface{}{"server": server}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
