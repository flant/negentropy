package authz

import (
	"errors"
	"fmt"
	"net/mail"

	"github.com/hashicorp/go-hclog"
	hcapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	uuidlib "github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	authn2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type Authorizator struct {
	UserRepo   *iam_repo.UserRepository
	SaRepo     *iam_repo.ServiceAccountRepository
	EntityRepo *repo.EntityRepo
	EaRepo     *repo.EntityAliasRepo

	IdentityApi   *api.IdentityAPI
	MountAccessor *vault.MountAccessorGetter

	Logger hclog.Logger
}

func NewAutorizator(txn *io.MemoryStoreTxn, vaultClient *hcapi.Client, aGetter *vault.MountAccessorGetter, logger hclog.Logger) *Authorizator {
	return &Authorizator{
		Logger: logger.Named("AuthoriZator"),

		SaRepo:   iam_repo.NewServiceAccountRepository(txn),
		UserRepo: iam_repo.NewUserRepository(txn),

		EaRepo:     repo.NewEntityAliasRepo(txn),
		EntityRepo: repo.NewEntityRepo(txn),

		MountAccessor: aGetter,
		IdentityApi:   api.NewIdentityAPI(vaultClient, logger.Named("LoginIdentityApi")),
	}
}

func (a *Authorizator) Authorize(authnResult *authn2.Result, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	var user *iam.User
	var err error
	var uuid string

	// try to use universal check
	switch {
	case uuidlib.IsValid(authnResult.UUID):
		uuid = authnResult.UUID
		a.Logger.Debug(fmt.Sprintf("Start authz for uuid %s", uuid))
		user, err = a.UserRepo.GetByID(uuid)
	case isValidEmailAddress(authnResult.UUID):
		email := authnResult.UUID
		a.Logger.Debug(fmt.Sprintf("Start authz for email %s", email))
		user, err = a.UserRepo.GetByEmail(email)
	default:
		return nil, fmt.Errorf("recieved uuid is neither pure uiid nor email")
	}

	if err != nil && !errors.Is(err, iam.ErrNotFound) {
		return nil, err
	}

	var authzRes *logical.Auth
	var fullID string

	if user != nil {
		fullID = user.FullIdentifier
		a.Logger.Debug(fmt.Sprintf("Found user %s for %s", fullID, authnResult.UUID))
		authzRes, err = a.authorizeUser(user, method, source)
		uuid = user.UUID
	} else {
		// not found user try to found service account
		uuid = authnResult.UUID
		a.Logger.Debug(fmt.Sprintf("Not found user for %s. Try find service account", uuid))
		var sa *iam.ServiceAccount
		sa, err = a.SaRepo.GetByID(uuid)
		if err != nil && errors.Is(err, iam.ErrNotFound) {
			return nil, fmt.Errorf("not found iam entity %s", uuid)
		}
		if err != nil {
			return nil, err
		}

		fullID = sa.FullIdentifier

		a.Logger.Debug(fmt.Sprintf("Found service account %s for %s uuid", fullID, authnResult.UUID))
		authzRes, err = a.authorizeServiceAccount(sa, method, source)
	}

	if err != nil {
		return nil, err
	}

	if authzRes == nil {
		a.Logger.Warn(fmt.Sprintf("Nil autzRes %s", uuid))
		return nil, fmt.Errorf("not authz %s", uuid)
	}

	a.Logger.Debug(fmt.Sprintf("Start getting vault entity and entity alias %s", fullID))
	vaultAlias, entityId, err := a.getAlias(uuid, source)
	if err != nil {
		return nil, err
	}

	a.Logger.Debug(fmt.Sprintf("Got entityId %s and entity alias %s", entityId, vaultAlias.ID))

	authzRes.Alias = vaultAlias
	authzRes.EntityID = entityId

	method.PopulateTokenAuth(authzRes)
	authzRes.InternalData["flantIamAuthMethod"] = method.Name

	a.Logger.Debug(fmt.Sprintf("Token auth populated %s", fullID))

	a.populateAuthnData(authzRes, authnResult)

	a.Logger.Debug(fmt.Sprintf("Authn data populated %s", fullID))

	return authzRes, nil
}

func (a *Authorizator) authorizeServiceAccount(sa *iam.ServiceAccount, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	// todo some logic for sa here
	return &logical.Auth{
		DisplayName:  sa.FullIdentifier,
		InternalData: map[string]interface{}{},
	}, nil
}

func (a *Authorizator) authorizeUser(user *iam.User, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	// todo some logic for user here
	return &logical.Auth{
		DisplayName:  user.FullIdentifier,
		InternalData: map[string]interface{}{},
	}, nil
}

func (a *Authorizator) populateAuthnData(authzRes *logical.Auth, authnResult *authn2.Result) {
	if len(authnResult.Metadata) > 0 {
		for k, v := range authnResult.Metadata {
			if _, ok := authzRes.Metadata[k]; ok {
				a.Logger.Warn(fmt.Sprintf("Key %s already exists in authz metadata. Skip", k))
				continue
			}

			authzRes.Metadata[k] = v
		}
	}

	if len(authnResult.InternalData) > 0 {
		for k, v := range authnResult.InternalData {
			if _, ok := authzRes.Metadata[k]; ok {
				a.Logger.Warn(fmt.Sprintf("Key %s already exists in authz internal data. Skip", k))
				continue
			}

			authzRes.InternalData[k] = v
		}
	}

	if len(authnResult.Policies) > 0 {
		authzRes.Policies = append(authzRes.Policies, authnResult.Policies...)
	}

	if len(authnResult.GroupAliases) > 0 {
		for i, p := range authnResult.GroupAliases {
			k := fmt.Sprintf("group_alias_%v", i)
			authzRes.Metadata[k] = p
		}
	}
}

func (a *Authorizator) getAlias(uuid string, source *model.AuthSource) (*logical.Alias, string, error) {
	entity, err := a.EntityRepo.GetByUserId(uuid)
	if err != nil {
		return nil, "", err
	}

	if entity == nil {
		return nil, "", fmt.Errorf("not found entity for %s", uuid)
	}

	a.Logger.Debug(fmt.Sprintf("Got entity from db %s", uuid))

	ea, err := a.EaRepo.GetForUser(uuid, source)
	if err != nil {
		return nil, "", err
	}

	if ea == nil {
		return nil, "", fmt.Errorf("not found entity alias for %s with source %s", uuid, source.Name)
	}

	a.Logger.Debug(fmt.Sprintf("Got entity alias from db %s", uuid))

	entityId, err := a.IdentityApi.EntityApi().GetID(entity.Name)
	if err != nil {
		return nil, "", fmt.Errorf("getting entity_id:%w", err)
	}

	if entityId == "" {
		return nil, "", fmt.Errorf("can not get entity id %s/%s", uuid, entity.Name)
	}

	a.Logger.Debug(fmt.Sprintf("Got entity id from vault %s", uuid))

	accessorId, err := a.MountAccessor.MountAccessor()
	if err != nil {
		return nil, "", err
	}

	eaId, err := a.IdentityApi.AliasApi().FindAliasIDByName(ea.Name, accessorId)
	if err != nil {
		return nil, "", err
	}

	if eaId == "" {
		return nil, "", fmt.Errorf("can not get entity alias id %s/%s/%s", uuid, ea.Name, source.Name)
	}

	a.Logger.Debug(fmt.Sprintf("Got entity alias id from db %s", uuid))

	return &logical.Alias{
		ID:            eaId,
		MountAccessor: accessorId,
		Name:          ea.Name,
	}, entityId, nil
}

func isValidEmailAddress(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
