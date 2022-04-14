package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
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
	UserRepo      *iam_repo.UserRepository
	SaRepo        *iam_repo.ServiceAccountRepository
	EntityRepo    *repo.EntityRepo
	EaRepo        *repo.EntityAliasRepo
	RoleRepo      *iam_repo.RoleRepository
	RolesResolver iam_usecase.RoleResolver

	MountAccessor *vault.MountAccessorGetter

	Logger              hclog.Logger
	vaultClientProvider client.VaultClientController
}

// Subject is a representation of iam.ServiceAccount or iam.User
type Subject struct {
	// user or service_account
	Type string `json:"type"`
	// UUID of iam.ServiceAccount or iam.User
	UUID string `json:"uuid"`
	//  tenant_uuid of subject
	TenantUUID iam.TenantUUID `json:"tenant_uuid"`
}

func MakeSubject(data map[string]interface{}) Subject {
	subjectType, _ := data["type"].(string)
	uuid, _ := data["uuid"].(string)
	tenantUUID, _ := data["tenant_uuid"].(string)
	return Subject{
		Type:       subjectType,
		UUID:       uuid,
		TenantUUID: tenantUUID,
	}
}

func NewAutorizator(txn *io.MemoryStoreTxn, vaultClientController client.VaultClientController, aGetter *vault.MountAccessorGetter, logger hclog.Logger) *Authorizator {
	return &Authorizator{
		Logger: logger.Named("AuthoriZator"),

		SaRepo:   iam_repo.NewServiceAccountRepository(txn),
		UserRepo: iam_repo.NewUserRepository(txn),

		EaRepo:     repo.NewEntityAliasRepo(txn),
		EntityRepo: repo.NewEntityRepo(txn),
		RoleRepo:   iam_repo.NewRoleRepository(txn),

		RolesResolver: iam_usecase.NewRoleResolver(txn),

		MountAccessor:       aGetter,
		vaultClientProvider: vaultClientController,
	}
}

func (a *Authorizator) identityApi() *api.IdentityAPI {
	return api.NewIdentityAPI(a.vaultClientProvider, a.Logger.Named("LoginIdentityApi"))
}

func (a *Authorizator) Authorize(authnResult *authn2.Result, method *model.AuthMethod, source *model.AuthSource,
	roleClaims []RoleClaim) (*logical.Auth, error) {
	subjectUUID := authnResult.UUID
	a.Logger.Debug(fmt.Sprintf("Start authz for %s", subjectUUID))

	authzRes, fullId, err := a.authorizeTokenOwner(subjectUUID, method, source)
	if err != nil {
		return nil, err
	}

	if authzRes == nil {
		a.Logger.Warn(fmt.Sprintf("Nil autzRes %s", subjectUUID))
		return nil, fmt.Errorf("not authz %s", subjectUUID)
	}

	a.Logger.Debug(fmt.Sprintf("Start getting vault entity and entity alias %s", fullId))
	vaultAlias, entityId, err := a.getAlias(subjectUUID, source)
	if err != nil {
		return nil, err
	}

	a.Logger.Debug(fmt.Sprintf("Got entityId %s and entity alias %s", entityId, vaultAlias.ID))

	authzRes.Alias = vaultAlias
	authzRes.EntityID = entityId
	subject := authzRes.InternalData["subject"].(Subject)

	method.PopulateTokenAuth(authzRes)

	err = a.addDynamicPolicies(authzRes, roleClaims, subject)
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

func (a *Authorizator) addDynamicPolicies(authzRes *logical.Auth, roleClaims []RoleClaim, subject Subject) error {
	extraPolicies := a.buildVaultPolicies(roleClaims, subject)
	if err := a.createDynamicPolicies(extraPolicies); err != nil {
		return err
	}
	for _, p := range extraPolicies {
		authzRes.Policies = append(authzRes.Policies, p.Name)
	}
	return nil
}

func (a *Authorizator) buildVaultPolicies(roleClaims []RoleClaim, subject Subject) []VaultPolicy {
	var result []VaultPolicy
	for _, rc := range roleClaims {
		policy := a.buildVaultPolicy(subject, rc)
		if policy != nil {
			result = append(result, *policy)
		}
	}
	return result
}

func (a *Authorizator) buildVaultPolicy(subject Subject, rc RoleClaim) *VaultPolicy {
	if rc.TenantUUID == "" {
		rc.TenantUUID = subject.TenantUUID
	}

	var policy VaultPolicy

	switch {
	case rc.Role == "ssh":
		role, err := a.RoleRepo.GetByID(rc.Role)
		if err != nil {
			a.Logger.Error("error catching role", "rolename", *role, "error", err)
			return nil
		}
		var effectiveRoles []iam_usecase.EffectiveRole
		var found bool
		switch {
		case subject.Type == "user" && role.Scope == "project":
			found, effectiveRoles, err = a.RolesResolver.CheckUserForProjectScopedRole(subject.UUID, rc.Role, rc.ProjectUUID)
		case subject.Type == "user" && role.Scope == "tenant":
			found, effectiveRoles, err = a.RolesResolver.CheckUserForTenantScopedRole(subject.UUID, rc.Role, rc.TenantUUID)
		case subject.Type == "service_account" && role.Scope == "project":
			found, effectiveRoles, err = a.RolesResolver.CheckServiceAccountForProjectScopedRole(subject.UUID, rc.Role,
				rc.ProjectUUID)
		case subject.Type == "service_account" && role.Scope == "tenant":
			found, effectiveRoles, err = a.RolesResolver.CheckServiceAccountForTenantScopedRole(subject.UUID, rc.Role,
				rc.TenantUUID)
		}
		if !found {
			a.Logger.Error("not found rolebindings", "subject_type", subject.Type, "subject_uuid",
				subject.UUID, "rolename", *role)
			return nil
		}
		if err != nil {
			a.Logger.Error("error searching rolebindings", "subject_type", subject.Type, "subject_uuid",
				subject.UUID, "rolename", *role, "error", err)
			return nil
		}
		ctx := context.Background()
		regoClaims := map[string]interface{}{
			"role":         rc.Role,
			"tenant_uuid":  rc.TenantUUID,
			"project_uuid": rc.ProjectUUID,
		}
		for k, v := range rc.Claim {
			regoClaims[k] = v
		}
		regoResult, err := ApplyRegoPolicy(ctx, sshPolicy1, UserData{}, effectiveRoles, regoClaims)
		if err != nil {
			a.Logger.Error(fmt.Sprintf("err:%s", err.Error()))
		} else {
			a.Logger.Debug(fmt.Sprintf("regoResult:%#v\n", *regoResult))
		}
		if !regoResult.Allow {
			a.Logger.Warn("not allowed", "subject_type", subject.Type, "subject_uuid",
				subject.UUID, "rolename", *role, "claims", rc)
			return nil
		}
		policy = VaultPolicy{
			Name:  fmt.Sprintf("%s_by_%s", rc.Role, subject.UUID),
			Rules: regoResult.VaultRules,

			// []Rule{
			// {
			//	Path:   "ssh/sign/signer",
			//	Update: true,
			// }, {
			//	Path: "auth/flant_iam_auth/multipass_owner",
			//	Read: true,
			// }, {
			//	Path: "auth/flant_iam_auth/query_server", // TODO
			//	Read: true,
			// }, {
			//	Path: "auth/flant_iam_auth/tenant/*", // TODO  split for tenant_list and others
			//	Read: true,
			//	List: true,
			// },
			// },
		}
		a.Logger.Debug("REMOVE IT VaultPolicy= %#v", policy)

	case rc.Role == "iam_read" && rc.TenantUUID != "":
		policy = VaultPolicy{
			Name: fmt.Sprintf("%s_tenant_%s_by_%s", rc.Role, rc.TenantUUID, subject.UUID),
			Rules: []Rule{{
				Path: "flant_iam/tenant/" + rc.TenantUUID + "*",
				Read: true,
				List: true,
			}, {
				Path: "flant_iam/role/*",
				Read: true,
				List: true,
			}, {
				Path: "flant_iam/feature_flag/*",
				Read: true,
				List: true,
			}},
		}

	case rc.Role == "iam_write" && rc.TenantUUID != "":
		policy = VaultPolicy{
			Name: fmt.Sprintf("%s_tenant_%s_by_%s", rc.Role, rc.TenantUUID, subject.UUID),
			Rules: []Rule{{
				Path:   "flant_iam/tenant/" + rc.TenantUUID + "*",
				Read:   true,
				List:   true,
				Create: true,
				Update: true,
				Delete: true,
			}, {
				Path: "flant_iam/role/*",
				Read: true,
				List: true,
			}, {
				Path: "flant_iam/feature_flag/*",
				Read: true,
				List: true,
			}},
		}

	case rc.Role == "iam_read_all":
		policy = VaultPolicy{
			Name: fmt.Sprintf("%s_by_%s", rc.Role, subject.UUID),
			Rules: []Rule{{
				Path: "flant_iam/*",
				Read: true,
				List: true,
			}, {
				Path: "flant_iam/role/*",
				Read: true,
				List: true,
			}, {
				Path: "flant_iam/feature_flag/*",
				Read: true,
				List: true,
			}},
		}

	case rc.Role == "iam_write_all":
		policy = VaultPolicy{
			Name: fmt.Sprintf("%s_by_%s", rc.Role, subject.UUID),
			Rules: []Rule{{
				Path:   "flant_iam/*",
				Read:   true,
				List:   true,
				Create: true,
				Update: true,
				Delete: true,
			}, {
				Path: "flant_iam/role/*",
				Read: true,
				List: true,
			}, {
				Path: "flant_iam/feature_flag/*",
				Read: true,
				List: true,
			}},
		}

	case rc.Role == "servers":
		policy = VaultPolicy{
			Name: fmt.Sprintf("%s_by_%s", rc.Role, subject.UUID),
			Rules: []Rule{{
				Path: "auth/flant_iam_auth/tenant/*",
				Read: true,
				List: true,
			}},
		}

	case rc.Role == "register_server" && rc.TenantUUID != "" && rc.ProjectUUID != "":
		policy = VaultPolicy{
			Name: fmt.Sprintf("%s_at_project_%s_of_%s_by_%s", rc.Role, rc.ProjectUUID, rc.TenantUUID, subject.UUID),
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
		policy = VaultPolicy{
			Name: fmt.Sprintf("%s_tenant_%s_by_%s", rc.Role, rc.TenantUUID, subject.UUID),
			Rules: []Rule{{
				Path: "auth/flant_iam_auth/tenant/" + rc.TenantUUID + "*",
				Read: true,
				List: true,
			}},
		}

	case rc.Role == "flow_read":
		policy = VaultPolicy{
			Name: fmt.Sprintf("flow_read_by_%s", subject.UUID),
			Rules: []Rule{{
				Path: "flant_iam/client/*",
				Read: true,
				List: true,
			}, {
				Path: "flant_iam/team/*",
				Read: true,
				List: true,
			}},
		}

	case rc.Role == "flow_write":
		policy = VaultPolicy{
			Name: fmt.Sprintf("flow_write_by_%s", subject.UUID),
			Rules: []Rule{{
				Path:   "flant_iam/client",
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
				List:   true,
			}, {
				Path:   "flant_iam/client/*",
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
				List:   true,
			}, {
				Path:   "flant_iam/team",
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
				List:   true,
			}, {
				Path:   "flant_iam/team/*",
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
				List:   true,
			}},
		}
	}
	return &policy
}

// authorizeServiceAccount called from authorizeTokenOwner in case token is owned by service_account
func (a *Authorizator) authorizeServiceAccount(sa *iam.ServiceAccount, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	// todo some logic for sa here
	// todo collect rba for user
	return &logical.Auth{
		DisplayName: sa.FullIdentifier,
		InternalData: map[string]interface{}{
			"subject": Subject{
				Type:       "service_account",
				UUID:       sa.UUID,
				TenantUUID: sa.TenantUUID,
			},
		},
	}, nil
}

// authorizeUser called from authorizeTokenOwner in case token is owned by user
func (a *Authorizator) authorizeUser(user *iam.User, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	// todo some logic for user here
	// todo collect rba for user
	return &logical.Auth{
		DisplayName: user.FullIdentifier,
		InternalData: map[string]interface{}{
			"subject": Subject{
				Type:       "user",
				UUID:       user.UUID,
				TenantUUID: user.TenantUUID,
			},
		},
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

func (a *Authorizator) createDynamicPolicies(policies []VaultPolicy) error {
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

func (a *Authorizator) Renew(method *model.AuthMethod, auth *logical.Auth, txn *io.MemoryStoreTxn, subject Subject) (*logical.Auth, error) {
	// check is user/sa still active
	// TODO check rolebinding still active
	a.Logger.Debug(fmt.Sprintf("===REMOVE IT !!!!=== %#v", subject))
	var owner memdb.Archivable
	var err error
	switch subject.Type {
	case iam.UserType:
		owner, err = iam_repo.NewUserRepository(txn).GetByID(subject.UUID)
	case iam.ServiceAccountType:
		owner, err = iam_repo.NewServiceAccountRepository(txn).GetByID(subject.UUID)
	default:
		return nil, fmt.Errorf("wrong type of tokenOwnerType:%s", subject.Type)
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

// TODo REMOVE IT! ==========================
var sshPolicy1 = `
package negentropy


default requested_ttl = "600s"
default requested_max_ttl = "1200s"

requested_ttl = input.ttl
requested_max_ttl = input.max_ttl

filtered_bindings[r] {
#	tenant := input.tenant_uuid
    project := input.project_uuid
	some i
	r := data.effective_roles[i]
#   	data.effective_roles[i].tenant_uuid==tenant
    	data.effective_roles[i].projects[_]==project
        to_seconds_number(data.effective_roles[i].options.ttl)>=to_seconds_number(requested_ttl)
        to_seconds_number(data.effective_roles[i].options.max_ttl)>=to_seconds_number(requested_max_ttl)
}

default allow = false

allow {count(filtered_bindings) >0}

# пути по которым должен появится доступ
rules = [
	{"path":"ssh/sign/signer","capabilities":["update"]}
    ]{allow}

ttl := requested_ttl {allow}

max_ttl := requested_max_ttl {allow}

# Переводим в число секунд
to_seconds_number(t) = x {
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
	 x = value ; endswith(lower_t, "s")
}{
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
	 x = value*60 ; endswith(lower_t, "m")
}{
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
     x = value*3600 ; endswith(lower_t, "h")
}`

// TODo REMOVE IT! ==========================
