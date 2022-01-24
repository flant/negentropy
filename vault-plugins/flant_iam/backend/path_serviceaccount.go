package backend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/sethvargo/go-password/password"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type serviceAccountBackend struct {
	logical.Backend
	storage         *io.MemoryStore
	tokenController *jwt.Controller
}

func serviceAccountPaths(b logical.Backend, tokenController *jwt.Controller, storage *io.MemoryStore) []*framework.Path {
	bb := &serviceAccountBackend{
		Backend:         b,
		storage:         storage,
		tokenController: tokenController,
	}
	return bb.paths()
}

func (b serviceAccountBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Service account creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "CIDRs",
					Required:    true,
				},
				"token_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
				"token_max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create serviceAccount.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create serviceAccount.",
				},
			},
		},
		// Service account creation with known uuid in advance
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a serviceAccount",
					Required:    true,
				},
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "CIDRs",
					Required:    true,
				},
				"token_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
				"token_max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create serviceAccount with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create serviceAccount with preexistent ID.",
				},
			},
		},
		// Service account list
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/?",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived groups",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all serviceAccounts IDs.",
				},
			},
		},
		// Service account by uuid: read, update, delete
		{

			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a serviceAccount",
					Required:    true,
				},
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
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
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "CIDRs",
					Required:    true,
				},
				"token_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
				"token_max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the service account by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the service account by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the service account by ID.",
				},
			},
		},

		// Multipass creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/" + uuid.Pattern("owner_uuid") + "/multipass",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of the tenant service account",
					Required:    true,
				},
				"ttl": {
					Type:        framework.TypeInt,
					Description: "TTL in seconds",
					Required:    true,
				},
				"max_ttl": {
					Type:        framework.TypeInt,
					Description: "Max TTL in seconds",
					Required:    true,
				},
				"description": {
					Type:        framework.TypeString,
					Description: "The purpose of issuing",
					Required:    true,
				},
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Allowed CIDRs to use the multipass from",
					Required:    true,
				},
				"allowed_roles": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Allowed roles to use the multipass with",
					Required:    true,
				},
			},
			ExistenceCheck: neverExisting,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleMultipassCreate(),
					Summary:  "Create service account multipass.",
				},
			},
		},
		// Multipass read or delete
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/" + uuid.Pattern("owner_uuid") + "/multipass/" + uuid.Pattern("uuid"),
			Fields: map[string]*framework.FieldSchema{

				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of the tenant service account",
					Required:    true,
				},
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a multipass",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleMultipassRead(),
					Summary:  "Get multipass by ID",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleMultipassDelete(),
					Summary:  "Delete multipass by ID",
				},
			},
		},
		// Multipass list
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/" + uuid.Pattern("owner_uuid") + "/multipass/?",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of the tenant service account",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived groups",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleMultipassList(),
					Summary:  "List multipass IDs",
				},
			},
		},

		// Password creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/" + uuid.Pattern("owner_uuid") + "/password",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of the tenant service account",
					Required:    true,
				},
				"description": {
					Type:        framework.TypeString,
					Description: "A comment or humans",
					Required:    true,
				},
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Allowed CIDRs to use the password from",
					Required:    true,
				},
				"allowed_roles": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Allowed roles to use the password with",
					Required:    true,
				},
				"ttl": {
					Type:        framework.TypeInt,
					Description: "TTL in seconds",
					Required:    true,
				},
			},
			ExistenceCheck: neverExisting,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handlePasswordCreate(),
					Summary:  "Create service account password",
				},
			},
		},
		// Password read or delete
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/" + uuid.Pattern("owner_uuid") + "/password/" + uuid.Pattern("uuid"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of the tenant service account",
					Required:    true,
				},
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a password",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handlePasswordRead(),
					Summary:  "Get password by ID",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handlePasswordDelete(),
					Summary:  "Delete password by ID",
				},
			},
		},
		// Password list
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/" + uuid.Pattern("owner_uuid") + "/password/?",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of the tenant service account",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived passwords",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handlePasswordList(),
					Summary:  "List password IDs",
				},
			},
		},
	}
}

func errExistenseVerdict(err error) (bool, error) {
	if err == consts.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (b *serviceAccountBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		var (
			id       = data.Get("uuid").(string)
			tenantID = data.Get(iam_repo.TenantForeignPK).(string)
		)

		b.Logger().Debug("checking serviceAccount existence", "path", req.Path, "id", id, "op", req.Operation)
		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)

		_, err := usecase.ServiceAccounts(tx, consts.OriginIAM, tenantID).GetByID(id)
		return errExistenseVerdict(err)
	}
}

func (b *serviceAccountBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create service_account", "path", req.Path)
		var (
			id, err    = backentutils.GetCreationID(expectID, data)
			tenantUUID = data.Get(iam_repo.TenantForeignPK).(string)

			ttl    = data.Get("token_ttl").(int)
			maxttl = data.Get("token_max_ttl").(int)
		)

		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}

		serviceAccount := &model.ServiceAccount{
			UUID:        id,
			TenantUUID:  tenantUUID,
			BuiltinType: "",
			Identifier:  data.Get("identifier").(string),
			CIDRs:       data.Get("allowed_cidrs").([]string),
			TokenTTL:    time.Duration(ttl),
			TokenMaxTTL: time.Duration(maxttl),
			Origin:      consts.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err = usecase.ServiceAccounts(tx, consts.OriginIAM, tenantUUID).Create(serviceAccount); err != nil {
			msg := "cannot create service account"
			b.Logger().Error(msg, "err", err.Error())
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"service_account": serviceAccount}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *serviceAccountBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("update service_account", "path", req.Path)
		var (
			id         = data.Get("uuid").(string)
			tenantUUID = data.Get(iam_repo.TenantForeignPK).(string)

			ttl    = data.Get("token_ttl").(int)
			maxttl = data.Get("token_max_ttl").(int)
		)

		serviceAccount := &model.ServiceAccount{
			UUID:        id,
			TenantUUID:  data.Get(iam_repo.TenantForeignPK).(string),
			Version:     data.Get("resource_version").(string),
			Identifier:  data.Get("identifier").(string),
			BuiltinType: "",
			CIDRs:       data.Get("allowed_cidrs").([]string),
			TokenTTL:    time.Duration(ttl),
			TokenMaxTTL: time.Duration(maxttl),
			Origin:      consts.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.ServiceAccounts(tx, consts.OriginIAM, tenantUUID).Update(serviceAccount)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"service_account": serviceAccount}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serviceAccountBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete service_account", "path", req.Path)
		var (
			id         = data.Get("uuid").(string)
			tenantUUID = data.Get(iam_repo.TenantForeignPK).(string)
		)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.ServiceAccounts(tx, consts.OriginIAM, tenantUUID).Delete(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *serviceAccountBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read service_account", "path", req.Path)
		var (
			id         = data.Get("uuid").(string)
			tenantUUID = data.Get(iam_repo.TenantForeignPK).(string)
		)

		tx := b.storage.Txn(false)

		serviceAccount, err := usecase.ServiceAccounts(tx, consts.OriginIAM, tenantUUID).GetByID(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"service_account": serviceAccount}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serviceAccountBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list service_accounts", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}
		tenantUUID := data.Get(iam_repo.TenantForeignPK).(string)

		tx := b.storage.Txn(false)

		serviceAccounts, err := usecase.ServiceAccounts(tx, consts.OriginIAM, tenantUUID).List(showArchived)
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"service_accounts": serviceAccounts,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

// Multipass

func (b *serviceAccountBackend) handleMultipassCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create service_account multipass", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		// Check that the feature is available
		if err := isJwtEnabled(tx, b.tokenController); err != nil {
			return backentutils.ResponseErr(req, err)
		}

		var (
			tid  = data.Get("tenant_uuid").(string)
			said = data.Get("owner_uuid").(string)

			ttl         = time.Duration(data.Get("ttl").(int)) * time.Second
			maxTTL      = time.Duration(data.Get("max_ttl").(int)) * time.Second
			cidrs       = data.Get("allowed_cidrs").([]string)
			roles       = data.Get("allowed_roles").([]string)
			description = data.Get("description").(string)
		)

		issueFn := jwt.CreateIssueMultipassFunc(b.tokenController, tx)

		jwtString, multipass, err := usecase.
			ServiceAccountMultipasses(tx, consts.OriginIAM, tid, said).
			CreateWithJWT(issueFn, ttl, maxTTL, cidrs, roles, description)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"multipass": multipass,
			"token":     jwtString,
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *serviceAccountBackend) handleMultipassDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete service_account multipass", "path", req.Path)
		var (
			id   = data.Get("uuid").(string)
			tid  = data.Get("tenant_uuid").(string)
			said = data.Get("owner_uuid").(string)
		)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.ServiceAccountMultipasses(tx, consts.OriginIAM, tid, said).Delete(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}
		return logical.RespondWithStatusCode(&logical.Response{}, req, http.StatusNoContent)
	}
}

func (b *serviceAccountBackend) handleMultipassRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read service_account multipass", "path", req.Path)
		var (
			id  = data.Get("uuid").(string)
			tid = data.Get("tenant_uuid").(string)
			uid = data.Get("owner_uuid").(string)
		)
		tx := b.storage.Txn(false)

		mp, err := usecase.ServiceAccountMultipasses(tx, consts.OriginIAM, tid, uid).GetByID(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"multipass": iam_repo.OmitSensitive(mp)}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serviceAccountBackend) handleMultipassList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list service_account multipasses", "path", req.Path)
		tid := data.Get("tenant_uuid").(string)
		uid := data.Get("owner_uuid").(string)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}

		tx := b.storage.Txn(false)

		multipasses, err := usecase.ServiceAccountMultipasses(tx, consts.OriginIAM, tid, uid).PublicList(showArchived)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"multipasses": multipasses,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

// Password

func (b *serviceAccountBackend) handlePasswordCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create service_account multipass", "path", req.Path)
		var (
			ttl       = time.Duration(data.Get("ttl").(int)) * time.Second
			validTill = time.Now().Add(ttl).Unix()

			tenantUUID = data.Get("tenant_uuid").(string)
			ownerUUID  = data.Get("owner_uuid").(string)
		)

		pass := &model.ServiceAccountPassword{
			UUID:        uuid.New(),
			TenantUUID:  tenantUUID,
			OwnerUUID:   ownerUUID,
			Description: data.Get("description").(string),
			TTL:         ttl,
			ValidTill:   validTill,
			CIDRs:       data.Get("allowed_cidrs").([]string),
			Roles:       data.Get("allowed_roles").([]string),
		}

		var err error
		pass.Secret, err = generatePassword()
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err = usecase.ServiceAccountPasswords(tx, tenantUUID, ownerUUID).Create(pass)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		// Includes sensitive data here
		resp := &logical.Response{
			Data: map[string]interface{}{
				"password": pass,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *serviceAccountBackend) handlePasswordDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete service_account password", "path", req.Path)
		var (
			tenantUUID = data.Get("tenant_uuid").(string)
			ownerUUID  = data.Get("owner_uuid").(string)
			id         = data.Get("uuid").(string)
		)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.ServiceAccountPasswords(tx, tenantUUID, ownerUUID).Delete(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}
		return logical.RespondWithStatusCode(&logical.Response{}, req, http.StatusNoContent)
	}
}

func (b *serviceAccountBackend) handlePasswordRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read service_account multipass", "path", req.Path)
		var (
			tenantUUID = data.Get("tenant_uuid").(string)
			ownerUUID  = data.Get("owner_uuid").(string)
			id         = data.Get("uuid").(string)
		)

		tx := b.storage.Txn(false)

		pass, err := usecase.ServiceAccountPasswords(tx, tenantUUID, ownerUUID).GetByID(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"password": iam_repo.OmitSensitive(pass)}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *serviceAccountBackend) handlePasswordList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list service_account passwords", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}
		var (
			tenantUUID = data.Get("tenant_uuid").(string)
			ownerUUID  = data.Get("owner_uuid").(string)
		)

		tx := b.storage.Txn(false)

		passwords, err := usecase.ServiceAccountPasswords(tx, tenantUUID, ownerUUID).List(showArchived)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"passwords": passwords,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func generatePassword() (string, error) {
	// Generate a password that is 64 characters long with 10 digits, 10 symbols,
	// allowing upper and lower case letters, disallowing repeat characters.
	generatorInput := &password.GeneratorInput{
		LowerLetters: password.LowerLetters,
		UpperLetters: password.UpperLetters,
		Digits:       password.Digits,
		Symbols:      "~!@#$%^&*()_+`-={}|[]:<>?,./", // remove \ and " to escape double backslashes marshaling problem
	}
	passwordGenerator, err := password.NewGenerator(generatorInput)
	if err != nil {
		return "", err
	}
	return passwordGenerator.Generate(64, 10, 10, false, false)
}
