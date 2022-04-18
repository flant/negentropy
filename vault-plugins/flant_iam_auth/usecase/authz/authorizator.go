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
	PolicyRepo    *repo.PolicyRepository
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

		EaRepo:        repo.NewEntityAliasRepo(txn),
		EntityRepo:    repo.NewEntityRepo(txn),
		RoleRepo:      iam_repo.NewRoleRepository(txn),
		PolicyRepo:    repo.NewPolicyRepository(txn),
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

	err = a.addDynamicPolicies(authzRes, roleClaims, subject, method.Name)
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

func (a *Authorizator) addDynamicPolicies(authzRes *logical.Auth, roleClaims []RoleClaim, subject Subject, authMethod string) error {
	extraPolicies, err := a.buildVaultPolicies(roleClaims, subject, authMethod)
	if err != nil {
		return err
	}
	err = a.createDynamicPolicies(extraPolicies)
	if err != nil {
		return err
	}
	for _, p := range extraPolicies {
		authzRes.Policies = append(authzRes.Policies, p.Name)
	}
	return nil
}

func (a *Authorizator) buildVaultPolicies(roleClaims []RoleClaim, subject Subject, authMethod string) ([]VaultPolicy, error) {
	var result []VaultPolicy
	var err error
	for _, rc := range roleClaims {
		rc, err = a.checkOrFillTenantUUID(rc, subject)
		if err != nil {
			return nil, err
		}
		negentropyPolicy, err := a.seekAndValidatePolicy(rc.Role, authMethod)
		if err != nil {
			return nil, err
		}
		policy, err := a.buildVaultPolicy(negentropyPolicy.Rego, subject, rc)
		if err != nil {
			return nil, err
		}
		if policy != nil {
			result = append(result, *policy)
		}
	}
	return result, nil
}

func (a *Authorizator) buildVaultPolicy(regoPolicy string, subject Subject, rc RoleClaim) (*VaultPolicy, error) {
	if rc.TenantUUID == "" {
		rc.TenantUUID = subject.TenantUUID
	}

	var policy VaultPolicy

	switch {
	case rc.Role == "ssh":
		role, err := a.RoleRepo.GetByID(rc.Role)
		if err != nil {
			err = fmt.Errorf("error catching role %s:%w", rc.Role, err)
			a.Logger.Error(err.Error())
			return nil, err
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
			err = fmt.Errorf("not found rolebindings, subject_type=%s, subject_uuid=%s, role=%s",
				subject.Type, subject.UUID, role.Name)
			a.Logger.Error(err.Error())
			return nil, err
		}
		if err != nil {
			err = fmt.Errorf("error searching, subject_type=%s, subject_uuid=%s, role=%s :%w",
				subject.Type, subject.UUID, role.Name, err)
			a.Logger.Error(err.Error())
			return nil, err
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
		regoResult, err := ApplyRegoPolicy(ctx, regoPolicy, UserData{}, effectiveRoles, regoClaims)
		if err != nil {
			err = fmt.Errorf("error appliing rego policy:%w", err)
			a.Logger.Error(err.Error())
			return nil, err
		} else {
			a.Logger.Debug(fmt.Sprintf("regoResult:%#v\n", *regoResult))
		}
		if !regoResult.Allow {
			err = fmt.Errorf("not allowed: subject_type=%s, subject_uuid=%s, rolename=%s, claims=%v",
				subject.Type, subject.UUID, role.Name, rc)
			a.Logger.Error(err.Error())
			return nil, err
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
	return &policy, nil
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

// TODO REMOVE IT AFTER IMPLEMENT ALL:
var tmpNotSeekPoliciesRoles = map[string]struct{}{
	"iam_read":        {},
	"iam_write":       {},
	"iam_read_all":    {},
	"iam_write_all":   {},
	"servers":         {},
	"register_server": {},
	"iam_auth_read":   {},
	"flow_read":       {},
	"flow_write":      {},
}

func (a *Authorizator) seekAndValidatePolicy(roleName iam.RoleName, authMethod string) (*model.Policy, error) {
	if _, tmpSkip := tmpNotSeekPoliciesRoles[roleName]; tmpSkip { // TODO  tmp stub
		fmt.Printf("====================Attantion! ===========================\n")
		fmt.Printf("skipped searching negentropy policy for role %s\n", roleName)
		fmt.Printf("====================Attantion! ===========================\n")
		return &model.Policy{}, nil // TODO  tmp stub
	} // TODO  tmp stub
	policies, err := a.PolicyRepo.ListActiveForRole(roleName)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, fmt.Errorf("no one negentropy policy for role:%s", roleName)
	}
	if len(policies) > 1 {
		return nil, fmt.Errorf("more the one negentropy policy for role:%s", roleName)
	}
	policy := policies[0]

	if len(policy.AllowedAuthMethods) == 0 {
		return policy, nil
	}
	for _, m := range policy.AllowedAuthMethods {
		if m == authMethod {
			return policy, nil
		}
	}
	return nil, fmt.Errorf("for role:%s authMethod %s is not allowed", roleName, authMethod)
}

// checkOrFillTenantUUID check or fill tenantUUID un RoleClaim:
// if it filled - it checks is it owner of subject or is subject shared to this tenant
// if not filled - fill by  owner of subject
func (a *Authorizator) checkOrFillTenantUUID(rc RoleClaim, subject Subject) (RoleClaim, error) {
	if rc.TenantUUID == subject.TenantUUID {
		return rc, nil
	}
	if rc.TenantUUID == "" {
		rc.TenantUUID = subject.TenantUUID
		return rc, nil
	}
	var isShared bool
	var err error
	if subject.Type == "user" {
		isShared, err = a.RolesResolver.IsUserSharedWithTenant(subject.UUID, rc.TenantUUID)
	} else {
		isShared, err = a.RolesResolver.IsServiceAccountSharedWithTenant(subject.UUID, rc.TenantUUID)
	}
	if err != nil {
		return RoleClaim{}, err
	}
	if !isShared {
		return RoleClaim{}, fmt.Errorf("role_claim: %#v has invalid tenant_uuid", rc)
	}
	return rc, nil
}
