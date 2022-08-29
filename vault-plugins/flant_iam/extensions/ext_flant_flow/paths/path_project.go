package paths

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type projectBackend struct {
	*flantFlowExtension
}

func projectPaths(e *flantFlowExtension) []*framework.Path {
	bb := &projectBackend{
		flantFlowExtension: e,
	}
	return bb.paths()
}

func (b projectBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/project",
			Fields: map[string]*framework.FieldSchema{
				clientUUIDKey: {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for project",
					Required:    true,
				},
				"service_packs": {
					Type:          framework.TypeStringSlice,
					Description:   "Service packs",
					Required:      true,
					AllowedValues: model.AllowedServicePackNames,
				},
				"devops_team": {
					Type:        framework.TypeString,
					Description: "Devops team uuid, in case of passed devops_service_pack",
				},
				"internal_project_team": {
					Type:        framework.TypeString,
					Description: "Team uuid, in case of passed internal_project_service_pack",
				},
				"consulting_team": {
					Type:        framework.TypeString,
					Description: "Team uuid, in case of passed consulting_service_pack",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(false)),
					Summary:  "Create project.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(false)),
					Summary:  "Create project.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/project/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
				clientUUIDKey: {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"service_packs": {
					Type:          framework.TypeStringSlice,
					Description:   "Service packs",
					Required:      true,
					AllowedValues: model.AllowedServicePackNames,
				},
				"devops_team": {
					Type:        framework.TypeString,
					Description: "Devops team uuid, in case of passed devops_service_pack",
				},
				"internal_project_team": {
					Type:        framework.TypeString,
					Description: "Team uuid, in case of passed internal_project_service_pack",
				},
				"consulting_team": {
					Type:        framework.TypeString,
					Description: "Team uuid, in case of passed consulting_service_pack",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(true)),
					Summary:  "Create project with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(true)),
					Summary:  "Create project with preexistent ID.",
				},
			},
		},
		// List
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/project/?",
			Fields: map[string]*framework.FieldSchema{
				clientUUIDKey: {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived projects",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleList),
					Summary:  "Lists all projects IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/project/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
				clientUUIDKey: {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"service_packs": {
					Type:          framework.TypeStringSlice,
					Description:   "Service packs",
					Required:      true,
					AllowedValues: model.AllowedServicePackNames,
				},
				"devops_team": {
					Type:        framework.TypeString,
					Description: "Devops team uuid, in case of passed devops_service_pack",
				},
				"internal_project_team": {
					Type:        framework.TypeString,
					Description: "Team uuid, in case of passed internal_project_service_pack",
				},
				"consulting_team": {
					Type:        framework.TypeString,
					Description: "Team uuid, in case of passed consulting_service_pack",
				},
			},
			ExistenceCheck: b.handleExistence,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleUpdate),
					Summary:  "Update the project by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleRead),
					Summary:  "Retrieve the project by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleDelete),
					Summary:  "Deletes the project by ID.",
				},
			},
		},
		// Restore
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/project/" + uuid.Pattern("uuid") + "/restore" + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a user",
					Required:    true,
				},
				clientUUIDKey: {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleRestore),
					Summary:  "Restore the project by ID.",
				},
			},
		},
	}
}

func (b *projectBackend) handleExistence(_ context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	id := data.Get("uuid").(string)
	clientID := data.Get(clientUUIDKey).(string)
	b.Logger().Debug("checking project existence", "path", req.Path, "id", id, "op", req.Operation)

	if !uuid.IsValid(id) {
		return false, fmt.Errorf("id must be valid UUIDv4")
	}

	tx := b.storage.Txn(false)

	obj, err := usecase.Projects(tx, b.liveConfig).GetByID(id)
	if err != nil {
		return false, err
	}
	exists := obj != nil && obj.TenantUUID == clientID
	return exists, nil
}

func (b *projectBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create project", "path", req.Path)
		projectParams, err := getProjectParams(data, expectID, false)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		tx := b.storage.Txn(true)
		defer tx.Abort()

		var project *model.Project
		if project, err = usecase.Projects(tx, b.liveConfig).Create(*projectParams); err != nil {
			err = fmt.Errorf("cannot create project:%w", err)
			b.Logger().Error("error", "error", err.Error())
			return backentutils.ResponseErr(req, err)
		}

		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"project": project}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func getProjectParams(data *framework.FieldData, expectID, expectVersion bool) (*usecase.ProjectParams, error) {
	id, err := backentutils.GetCreationID(expectID, data)
	if err != nil {
		return nil, err
	}
	version := ""
	if expectVersion {
		version = data.Get("resource_version").(string)
	}
	servicePacks, err := getServicePacks(data)
	if err != nil {
		return nil, err
	}
	devopsTeamUUID := data.Get("devops_team").(model.TeamUUID)
	internalProjectTeamUUID := data.Get("internal_project_team").(model.TeamUUID)
	consultingTeamUUID := data.Get("consulting_team").(model.TeamUUID)
	return &usecase.ProjectParams{
		IamProject: &iam_model.Project{
			UUID:       id,
			TenantUUID: data.Get(clientUUIDKey).(string),
			Version:    version,
			Identifier: data.Get("identifier").(string),
		},
		ServicePackNames:        servicePacks,
		DevopsTeamUUID:          devopsTeamUUID,
		InternalProjectTeamUUID: internalProjectTeamUUID,
		ConsultingTeamUUID:      consultingTeamUUID,
	}, nil
}

// returns only unique service_packs names
func getServicePacks(data *framework.FieldData) (map[model.ServicePackName]struct{}, error) {
	servicePacksRaw := data.Get("service_packs")

	servicePacksArr, ok := servicePacksRaw.([]model.ServicePackName)
	if !ok {
		err := fmt.Errorf("marshalling params: wrong type of param service_packs, cant cast to []model.ServicePackName, passed value:%#v",
			servicePacksRaw)
		return nil, err
	}
	servicePacks := map[model.ServicePackName]struct{}{}
	for _, sp := range servicePacksArr {
		if _, ok = model.ServicePackNames[sp]; !ok {
			return nil, fmt.Errorf("%w: wrong service_pack name:%s", consts.ErrInvalidArg, sp)
		}
		servicePacks[sp] = struct{}{}
	}
	return servicePacks, nil
}

func (b *projectBackend) handleUpdate(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("update project", "path", req.Path)

	projectParams, err := getProjectParams(data, true, true)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	tx := b.storage.Txn(true)
	defer tx.Abort()

	project, err := usecase.Projects(tx, b.liveConfig).Update(*projectParams)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	if err = io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{"project": project}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *projectBackend) handleDelete(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("delete project", "path", req.Path)

	id := data.Get("uuid").(string)

	tx := b.storage.Txn(true)
	defer tx.Abort()

	err := usecase.Projects(tx, b.liveConfig).Delete(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
}

func (b *projectBackend) handleRead(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("read project", "path", req.Path)

	id := data.Get("uuid").(string)

	tx := b.storage.Txn(false)

	project, err := usecase.Projects(tx, b.liveConfig).GetByID(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{"project": project}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *projectBackend) handleList(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("list projects", "path", req.Path)
	var showArchived bool
	rawShowArchived, ok := data.GetOk("show_archived")
	if ok {
		showArchived = rawShowArchived.(bool)
	}
	clientID := data.Get(clientUUIDKey).(string)

	tx := b.storage.Txn(false)

	projects, err := usecase.Projects(tx, b.liveConfig).List(clientID, showArchived)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"projects": projects,
		},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *projectBackend) handleRestore(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("restore project", "path", req.Path)
	tx := b.storage.Txn(true)
	defer tx.Abort()

	id := data.Get("uuid").(string)

	project, err := usecase.Projects(tx, b.liveConfig).Restore(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"project": project,
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
