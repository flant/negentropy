package backend

import (
	"errors"
	"fmt"

	log "github.com/hashicorp/go-hclog"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	entity_api "github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/client"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type EntityIDOwner struct {
	// Expected value: `user` or `service_account`
	OwnerType string
	// Expected value type: iam.User or iam.ServiceAccount
	Owner interface{}
}

type EntityID = string

type EntityIDResolver interface {
	// RevealEntityIDOwner returns type and info about token owner by its EntityID
	// it can be iam.User, or iam.ServiceAccount
	RevealEntityIDOwner(EntityID, *io.MemoryStoreTxn) (*EntityIDOwner, error)
	// AvailableTenantsAndProjectsByEntityID returns sets of tenants and projects available for EntityID
	AvailableTenantsAndProjectsByEntityID(EntityID, *io.MemoryStoreTxn) (map[iam.TenantUUID]struct{}, map[iam.ProjectUUID]struct{}, error)
}

type entityIDResolver struct {
	logger      log.Logger
	vaultClient *client.VaultClientController // do not use  *entity_api.EntityAPI because vaultClient need successful Init before it can be used
}

func (r entityIDResolver) RevealEntityIDOwner(entityID EntityID, txn *io.MemoryStoreTxn) (*EntityIDOwner, error) {
	r.logger.Debug(fmt.Sprintf("EntityID=%s", entityID))
	vc, err := r.vaultClient.APIClient()
	if err != nil {
		return nil, fmt.Errorf("NewEntityIDResolver: internal error accessing vault client: %w", err)
	}

	entityApi := entity_api.NewIdentityAPI(vc, r.logger.Named("LoginIdentityApi")).EntityApi()

	ent, err := entityApi.GetByID(entityID)
	if err != nil {
		return nil, fmt.Errorf("finding vault entity by id: %w", err)
	}

	name, ok := ent["name"]
	if !ok {
		return nil, fmt.Errorf("field 'name' in vault entity is ommited")
	}

	nameStr, ok := name.(string)
	if !ok {
		return nil, fmt.Errorf("field 'name' should be string")
	}
	r.logger.Debug(fmt.Sprintf("catch name of vault entity: %s", nameStr))

	iamEntity, err := repo.NewEntityRepo(txn).GetByName(nameStr)
	if err != nil {
		return nil, fmt.Errorf("finding iam_entity by name:%w", err)
	}
	r.logger.Debug(fmt.Sprintf("catch multipass owner UUID: %s, try to find user", iamEntity.UserId))

	user, err := iam_repo.NewUserRepository(txn).GetByID(iamEntity.UserId)
	if err != nil && !errors.Is(err, iam.ErrNotFound) {
		return nil, fmt.Errorf("finding user by id:%w", err)
	}
	if err == nil {
		r.logger.Debug(fmt.Sprintf("found user UUID: %s", user.UUID))
		return &EntityIDOwner{
			OwnerType: iam.UserType,
			Owner:     user,
		}, nil
	} else {
		r.logger.Debug("Not found user, try to find service_account")
		sa, err := iam_repo.NewServiceAccountRepository(txn).GetByID(iamEntity.UUID)
		if err != nil && !errors.Is(err, iam.ErrNotFound) {
			return nil, fmt.Errorf("finding service_account by id:%w", err)
		}
		if errors.Is(err, iam.ErrNotFound) {
			r.logger.Debug("Not found neither user nor service_account")
			return nil, err
		}
		r.logger.Debug(fmt.Sprintf("found service_account UUID: %s", sa.UUID))
		return &EntityIDOwner{
			OwnerType: iam.ServiceAccountType,
			Owner:     sa,
		}, nil
	}
}

func (r entityIDResolver) AvailableTenantsAndProjectsByEntityID(entityID EntityID, txn *io.MemoryStoreTxn) (map[iam.TenantUUID]struct{},
	map[iam.ProjectUUID]struct{}, error) {
	entityIDOwner, err := r.RevealEntityIDOwner(entityID, txn)
	if errors.Is(err, iam.ErrNotFound) {
		return map[iam.TenantUUID]struct{}{}, map[iam.ProjectUUID]struct{}{}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	switch entityIDOwner.OwnerType {
	case iam.UserType:
		{
			user, ok := entityIDOwner.Owner.(*iam.User)
			if !ok {
				return nil, nil, fmt.Errorf("can't cast, need *model.User, got: %T", entityIDOwner.Owner)
			}
			groups, err := iam_repo.NewGroupRepository(txn).FindAllParentGroupsForUserUUID(user.UUID)
			gs := make([]iam.GroupUUID, 0, len(groups))
			for uuid := range groups {
				gs = append(gs, uuid)
			}
			if err != nil {
				return nil, nil, fmt.Errorf("collecting tenants, get groups: %w", err)
			}
			//  TODO Here easy to have two or three paralleled goroutines
			tenants, err := iam_repo.NewIdentitySharingRepository(txn).ListDestinationTenantsByGroupUUIDs(gs...)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting tenants, get target tenants: %w", err)
			}
			tenants[user.TenantUUID] = struct{}{}

			rbRepository := iam_repo.NewRoleBindingRepository(txn)
			userRBs, err := rbRepository.FindDirectRoleBindingsForUser(user.UUID)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects, get FindDirectRoleBindingsForUser: %w", err)
			}
			groupsRBs, err := rbRepository.FindDirectRoleBindingsForGroups(gs...)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects, get FindDirectRoleBindingsForGroups: %w", err)
			}
			projectRepo := iam_repo.NewProjectRepository(txn)
			projects, err := collectProjectUUIDsFromRoleBindings(userRBs, groupsRBs, projectRepo)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects: %w", err)
			}
			return tenants, projects, nil
		}

	case iam.ServiceAccountType:
		{
			sa, ok := entityIDOwner.Owner.(*iam.ServiceAccount)
			if !ok {
				return nil, nil, fmt.Errorf("can't cast, need *model.ServiceAccount, got: %T", entityIDOwner.Owner)
			}
			groups, err := iam_repo.NewGroupRepository(txn).FindAllParentGroupsForServiceAccountUUID(sa.UUID)
			gs := make([]iam.GroupUUID, 0, len(groups))
			for uuid := range groups {
				gs = append(gs, uuid)
			}
			if err != nil {
				return nil, nil, fmt.Errorf("collecting tenants, get groups: %w", err)
			}
			//  TODO Here easy to have two or three paralleled goroutines
			tenants, err := iam_repo.NewIdentitySharingRepository(txn).ListDestinationTenantsByGroupUUIDs(gs...)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting tenants, get target tenants: %w", err)
			}
			tenants[sa.TenantUUID] = struct{}{}
			rbRepository := iam_repo.NewRoleBindingRepository(txn)

			userRBs, err := rbRepository.FindDirectRoleBindingsForServiceAccount(sa.UUID)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects, get FindDirectRoleBindingsForServiceAccount: %w", err)
			}
			groupsRBs, err := rbRepository.FindDirectRoleBindingsForGroups(gs...)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects, get FindDirectRoleBindingsForGroups: %w", err)
			}

			projectRepo := iam_repo.NewProjectRepository(txn)
			projects, err := collectProjectUUIDsFromRoleBindings(userRBs, groupsRBs, projectRepo)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects: %w", err)
			}
			return tenants, projects, nil
		}
	}
	return nil, nil, fmt.Errorf("unexpected subjectType: `%s`", entityIDOwner.OwnerType)
}

func NewEntityIDResolver(logger log.Logger, vaultClient *client.VaultClientController) (EntityIDResolver, error) {
	return &entityIDResolver{
		logger:      logger.Named("EntityIDResolver"),
		vaultClient: vaultClient,
	}, nil
}

func collectProjectUUIDsFromRoleBindings(rbs1 map[iam.RoleBindingUUID]*iam.RoleBinding,
	rbs2 map[iam.RoleBindingUUID]*iam.RoleBinding, projectRepo *iam_repo.ProjectRepository) (map[iam.ProjectUUID]struct{}, error) {
	result := map[iam.ProjectUUID]struct{}{}

	fullTenants := map[iam.TenantUUID]struct{}{}
	for _, rb := range rbs1 {
		err := processRoleBinding(rb, &fullTenants, projectRepo, &result)
		if err != nil {
			return nil, err
		}
	}
	for _, rb := range rbs2 {
		err := processRoleBinding(rb, &fullTenants, projectRepo, &result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// processRoleBinding check rb and write to given pointers
func processRoleBinding(rb *iam.RoleBinding, fullTenants *map[iam.TenantUUID]struct{},
	projectRepo *iam_repo.ProjectRepository, result *map[iam.ProjectUUID]struct{}) error {
	if rb.AnyProject {
		if _, processedTenant := (*fullTenants)[rb.TenantUUID]; !processedTenant {
			(*fullTenants)[rb.TenantUUID] = struct{}{}
			allTenantProject, err := projectRepo.ListIDs(rb.TenantUUID, false)
			if err != nil {
				return err
			}
			for _, p := range allTenantProject {
				(*result)[p] = struct{}{}
			}
		}
	} else {
		for _, pUUID := range rb.Projects {
			(*result)[pUUID] = struct{}{}
		}
	}
	return nil
}
