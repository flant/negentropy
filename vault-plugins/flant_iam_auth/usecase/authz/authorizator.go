package authz

import (
	"errors"
	"fmt"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	authn2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
	"github.com/flant/negentropy/vault-plugins/shared/client"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type Authorizator struct {
	UserRepo   *iam_repo.UserRepository
	SaRepo     *iam_repo.ServiceAccountRepository
	EntityRepo *repo.EntityRepo
	EaRepo     *repo.EntityAliasRepo

	MountAccessor *vault.MountAccessorGetter

	Logger              hclog.Logger
	vaultClientProvider client.VaultClientController
}

func NewAutorizator(txn *io.MemoryStoreTxn, vaultClientController client.VaultClientController, aGetter *vault.MountAccessorGetter, logger hclog.Logger) *Authorizator {
	return &Authorizator{
		Logger: logger.Named("AuthoriZator"),

		SaRepo:   iam_repo.NewServiceAccountRepository(txn),
		UserRepo: iam_repo.NewUserRepository(txn),

		EaRepo:     repo.NewEntityAliasRepo(txn),
		EntityRepo: repo.NewEntityRepo(txn),

		MountAccessor:       aGetter,
		vaultClientProvider: vaultClientController,
	}
}

func (a *Authorizator) identityApi() *api.IdentityAPI {
	return api.NewIdentityAPI(a.vaultClientProvider, a.Logger.Named("LoginIdentityApi"))
}

func (a *Authorizator) Authorize(authnResult *authn2.Result, method *model.AuthMethod, source *model.AuthSource,
	roleClaims []RoleClaim) (*logical.Auth, error) {
	uuid := authnResult.UUID
	a.Logger.Debug(fmt.Sprintf("Start authz for %s", uuid))

	authzRes, fullId, err := a.authorizeTokenOwner(uuid, method, source)
	if err != nil {
		return nil, err
	}

	if authzRes == nil {
		a.Logger.Warn(fmt.Sprintf("Nil autzRes %s", uuid))
		return nil, fmt.Errorf("not authz %s", uuid)
	}

	a.Logger.Debug(fmt.Sprintf("Start getting vault entity and entity alias %s", fullId))
	vaultAlias, entityId, err := a.getAlias(uuid, source)
	if err != nil {
		return nil, err
	}

	a.Logger.Debug(fmt.Sprintf("Got entityId %s and entity alias %s", entityId, vaultAlias.ID))

	authzRes.Alias = vaultAlias
	authzRes.EntityID = entityId

	method.PopulateTokenAuth(authzRes)

	err = a.addDynamicPolicies(authzRes, roleClaims, uuid)
	if err != nil {
		return nil, err
	}

	authzRes.InternalData["flantIamAuthMethod"] = method.Name

	a.Logger.Debug(fmt.Sprintf("Token auth populated %s", fullId))

	a.populateAuthnData(authzRes, authnResult)

	a.Logger.Debug(fmt.Sprintf("Authn data populated %s", fullId))

	return authzRes, nil
}

func (a *Authorizator) authorizeTokenOwner(uuid string, method *model.AuthMethod, source *model.AuthSource) (authzRes *logical.Auth, tokenOwnerFullIdentifier string, err error) {
	user, err := a.UserRepo.GetByID(uuid)
	if err != nil && !errors.Is(err, consts.ErrNotFound) {
		return nil, "", err
	}
	if user != nil && user.NotArchived() {
		tokenOwnerFullIdentifier = user.FullIdentifier
		a.Logger.Debug(fmt.Sprintf("Found user %s for %s uuid", tokenOwnerFullIdentifier, uuid))
		authzRes, err = a.authorizeUser(user, method, source)
	} else {
		// not found user try to found service account
		a.Logger.Debug(fmt.Sprintf("Not found active user for %s uuid. Try find service account", uuid))
		var sa *iam.ServiceAccount
		sa, err = a.SaRepo.GetByID(uuid)
		if errors.Is(err, consts.ErrNotFound) || sa.Archived() {
			return nil, "", fmt.Errorf("not found active iam entity %s", uuid)
		}
		if err != nil {
			return nil, "", err
		}

		tokenOwnerFullIdentifier = sa.FullIdentifier

		a.Logger.Debug(fmt.Sprintf("Found service account %s for %s uuid", tokenOwnerFullIdentifier, uuid))
		authzRes, err = a.authorizeServiceAccount(sa, method, source)
	}
	return authzRes, tokenOwnerFullIdentifier, nil
}

func (a *Authorizator) addDynamicPolicies(authzRes *logical.Auth, roleClaims []RoleClaim, userUUID iam.UserUUID) error {
	extraPolicies := buildPolicies(roleClaims, userUUID)
	if err := a.createDynamicPolicies(extraPolicies); err != nil {
		return err
	}
	for _, p := range extraPolicies {
		authzRes.Policies = append(authzRes.Policies, p.Name)
	}
	return nil
}

func buildPolicies(roleClaims []RoleClaim, userUUID iam.UserUUID) []Policy {
	var result []Policy
	for _, rc := range roleClaims {
		var policy Policy
		switch {
		case rc.Role == "iam_read" && rc.TenantUUID != "":
			policy = Policy{
				Name: fmt.Sprintf("%s_tenant_%s_by_%s", rc.Role, rc.TenantUUID, userUUID),
				Rules: []Rule{{
					Path: "flant_iam/tenant/" + rc.TenantUUID + "*",
					Read: true,
					List: true,
				}},
			}

		case rc.Role == "iam_write" && rc.TenantUUID != "":
			policy = Policy{
				Name: fmt.Sprintf("%s_tenant_%s_by_%s", rc.Role, rc.TenantUUID, userUUID),
				Rules: []Rule{{
					Path:   "flant_iam/tenant/" + rc.TenantUUID + "*",
					Read:   true,
					List:   true,
					Create: true,
					Update: true,
					Delete: true,
				}},
			}

		case rc.Role == "iam_read_all":
			policy = Policy{
				Name: fmt.Sprintf("%s_by_%s", rc.Role, userUUID),
				Rules: []Rule{{
					Path: "flant_iam/*",
					Read: true,
					List: true,
				}},
			}

		case rc.Role == "iam_write_all":
			policy = Policy{
				Name: fmt.Sprintf("%s_by_%s", rc.Role, userUUID),
				Rules: []Rule{{
					Path:   "flant_iam/*",
					Read:   true,
					List:   true,
					Create: true,
					Update: true,
					Delete: true,
				}},
			}

		case rc.Role == "servers":
			policy = Policy{
				Name: fmt.Sprintf("%s_by_%s", rc.Role, userUUID),
				Rules: []Rule{{
					Path: "auth/flant_iam_auth/tenant/*",
					Read: true,
					List: true,
				}},
			}

		case rc.Role == "ssh":
			policy = Policy{
				Name: fmt.Sprintf("%s_by_%s", rc.Role, userUUID),
				Rules: []Rule{
					{
						Path:   "ssh/sign/signer",
						Update: true,
					}, {
						Path: "auth/flant_iam_auth/multipass_owner",
						Read: true,
					}, {
						Path: "auth/flant_iam_auth/query_server", // TODO
						Read: true,
					}, {
						Path: "auth/flant_iam_auth/tenant/*", // TODO  split for tenant_list and others
						Read: true,
						List: true,
					},
				},
			}

		case rc.Role == "register_server" && rc.TenantUUID != "" && rc.ProjectUUID != "":
			policy = Policy{
				Name: fmt.Sprintf("%s_at_project_%s_of_%s_by_%s", rc.Role, rc.ProjectUUID, rc.TenantUUID, userUUID),
				Rules: []Rule{
					{
						Path:   fmt.Sprintf("flant_iam/tenant/%s/project/%s/register_server*", rc.TenantUUID, rc.ProjectUUID),
						Create: true,
						Update: true,
					},
					{
						Path:   fmt.Sprintf("flant_iam/tenant/%s/project/%s/server*", rc.TenantUUID, rc.ProjectUUID),
						Create: true,
						Read:   true,
						Update: true,
						Delete: true,
						List:   true,
					},
				},
			}

		case rc.Role == "iam_auth_read" && rc.TenantUUID != "":
			policy = Policy{
				Name: fmt.Sprintf("%s_tenant_%s_by_%s", rc.Role, rc.TenantUUID, userUUID),
				Rules: []Rule{{
					Path: "auth/flant_iam_auth/tenant/" + rc.TenantUUID + "*",
					Read: true,
					List: true,
				}},
			}

		case rc.Role == "flow_read":
			policy = Policy{
				Name: fmt.Sprintf("flow_read_by_%s", userUUID),
				Rules: []Rule{{
					Path: "flant_flow/*",
					Read: true,
					List: true,
				}},
			}

		case rc.Role == "flow_write":
			policy = Policy{
				Name: fmt.Sprintf("flow_write_by_%s", userUUID),
				Rules: []Rule{{
					Path:   "flant_flow/*",
					Create: true,
					Read:   true,
					Update: true,
					Delete: true,
					List:   true,
				}},
			}
		}

		if policy.Name != "" {
			result = append(result, policy)
		}
	}
	return result
}

// authorizeServiceAccount called from authorizeTokenOwner in case token is owned by service_account
func (a *Authorizator) authorizeServiceAccount(sa *iam.ServiceAccount, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	// todo some logic for sa here
	// todo collect rba for user
	return &logical.Auth{
		DisplayName:  sa.FullIdentifier,
		InternalData: map[string]interface{}{"subject_type": "service_account", "subject_uuid": sa.UUID},
	}, nil
}

// authorizeUser called from authorizeTokenOwner in case token is owned by user
func (a *Authorizator) authorizeUser(user *iam.User, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	// todo some logic for user here
	// todo collect rba for user
	return &logical.Auth{
		DisplayName:  user.FullIdentifier,
		InternalData: map[string]interface{}{"subject_type": "user", "subject_uuid": user.UUID},
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

	entityId, err := a.identityApi().EntityApi().GetID(entity.Name)
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

	eaId, err := a.identityApi().AliasApi().FindAliasIDByName(ea.Name, accessorId)
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

func (a *Authorizator) createDynamicPolicies(policies []Policy) error {
	for _, p := range policies {
		err := backoff.Retry(func() error {
			client, err := a.vaultClientProvider.APIClient(nil)
			if err != nil {
				return err
			}
			return client.Sys().PutPolicy(p.Name, p.PolicyRules())
		}, io.FiveSecondsBackoff())
		if err != nil {
			return fmt.Errorf("put policy %s:%w", p.Name, err)
		}
	}
	return nil
}

func (a *Authorizator) Renew(method *model.AuthMethod, auth *logical.Auth, txn *io.MemoryStoreTxn,
	tokenOwnerType string, tokenOwnerUUID string) (*logical.Auth, error) {
	// check is user/sa still active
	var owner memdb.Archivable
	var err error
	switch tokenOwnerType {
	case iam.UserType:
		owner, err = iam_repo.NewUserRepository(txn).GetByID(tokenOwnerUUID)
	case iam.ServiceAccountType:
		owner, err = iam_repo.NewServiceAccountRepository(txn).GetByID(tokenOwnerUUID)
	default:
		return nil, fmt.Errorf("wrong type of tokenOwnerType:%s", tokenOwnerType)
	}
	if err != nil {
		return nil, err
	}
	if owner.Archived() {
		return nil, fmt.Errorf("tokenOwner is deleted")
	}
	authzRes := *auth
	authzRes.TTL = method.TokenTTL
	authzRes.MaxTTL = method.TokenMaxTTL
	authzRes.Period = method.TokenPeriod
	return &authzRes, nil
}
