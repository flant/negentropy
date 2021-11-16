package usecase

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type ServerService struct {
	tenantService      *usecase.TenantService
	projectsService    *usecase.ProjectService
	groupRepo          *iam_repo.GroupRepository
	serviceAccountRepo *iam_repo.ServiceAccountRepository
	roleService        *usecase.RoleService
	roleBindingRepo    *iam_repo.RoleBindingRepository
	serverRepo         *repo.ServerRepository

	tx *io.MemoryStoreTxn
}

func NewServerService(tx *io.MemoryStoreTxn) *ServerService {
	return &ServerService{
		tenantService:      usecase.Tenants(tx),
		projectsService:    usecase.Projects(tx),
		groupRepo:          iam_repo.NewGroupRepository(tx),
		serviceAccountRepo: iam_repo.NewServiceAccountRepository(tx),
		roleBindingRepo:    iam_repo.NewRoleBindingRepository(tx),
		serverRepo:         repo.NewServerRepository(tx),
		roleService:        usecase.Roles(tx),
		tx:                 tx,
	}
}

func (s *ServerService) Create(
	multipassIssue jwt.MultipassIssFn,
	tenantUUID, projectUUID, serverID string,
	labels, annotations map[string]string,
	roles []string,
) (string, string, error) {
	var (
		tenantBoundRoles  []iam_model.BoundRole
		projectBoundRoles []iam_model.BoundRole
	)

	server := &model.Server{
		UUID:        uuid.New(),
		TenantUUID:  tenantUUID,
		ProjectUUID: projectUUID,
		Version:     iam_repo.NewResourceVersion(),
		Identifier:  serverID,
		Labels:      labels,
		Annotations: annotations,
	}

	tenant, err := s.tenantService.GetByID(tenantUUID)
	if err != nil {
		return "", "", err
	}

	project, err := s.projectsService.GetByID(projectUUID)
	if err != nil {
		return "", "", err
	}

	rawServer, err := s.serverRepo.GetByID(tenantUUID, projectUUID, serverID)
	if err != nil && !errors.Is(err, iam_model.ErrNotFound) {
		return "", "", err
	}
	if rawServer != nil {
		return "", "", fmt.Errorf("server with identifier %q already exists in project %q", serverID, project.Identifier)
	}

	group, getGroupErr := s.groupRepo.GetByIdentifier(tenantUUID, nameForTenantLevelObjects(tenant.Identifier))
	if getGroupErr != nil && !errors.Is(getGroupErr, iam_model.ErrNotFound) {
		return "", "", err
	}
	if group == nil {
		group = &iam_model.Group{
			UUID:       uuid.New(),
			TenantUUID: tenant.UUID,
			Identifier: nameForTenantLevelObjects(tenant.Identifier),
			Origin:     iam_model.OriginServerAccess,
		}
	}

	// create RoleBinding for each role
	for _, roleName := range roles {
		role, err := s.roleService.Get(roleName)
		if err != nil {
			return "", "", err
		}

		switch role.Scope {
		case iam_model.RoleScopeTenant:
			tenantBoundRoles = append(tenantBoundRoles, iam_model.BoundRole{
				Name: role.Name,
			})
		case iam_model.RoleScopeProject:
			projectBoundRoles = append(projectBoundRoles, iam_model.BoundRole{
				Name: role.Name,
			})
		}
	}

	// FIXME: remove duplication
	if len(tenantBoundRoles) != 0 {
		var (
			roleBinding *iam_model.RoleBinding
			err         error
		)

		// TODO: update existing
		roleBinding, err = s.roleBindingRepo.GetByIdentifier(tenantUUID, nameForTenantLevelObjects(tenant.Identifier))
		if err != nil && !errors.Is(err, iam_model.ErrNotFound) {
			return "", "", err
		}

		if roleBinding == nil {
			newRoleBinding := &iam_model.RoleBinding{
				UUID:       uuid.New(),
				Version:    iam_repo.NewResourceVersion(),
				TenantUUID: tenant.ObjId(),
				Origin:     iam_model.OriginServerAccess,
				Identifier: nameForTenantLevelObjects(tenant.Identifier),
				Groups:     []iam_model.GroupUUID{group.UUID},
				Roles:      tenantBoundRoles,
			}

			err := s.roleBindingRepo.Create(newRoleBinding)
			if err != nil {
				return "", "", err
			}
		}
	}

	if len(projectBoundRoles) != 0 {
		var (
			roleBinding *iam_model.RoleBinding
			err         error
		)

		// TODO: update existing
		roleBinding, err = s.roleBindingRepo.GetByIdentifier(tenantUUID, nameForTenantLevelObjects(tenant.Identifier))
		if err != nil && !errors.Is(err, iam_model.ErrNotFound) {
			return "", "", err
		}

		if roleBinding == nil {
			newRoleBinding := &iam_model.RoleBinding{
				UUID:       uuid.New(),
				TenantUUID: tenant.ObjId(),
				Version:    iam_repo.NewResourceVersion(),
				Origin:     iam_model.OriginServerAccess,
				Identifier: nameForTenantLevelObjects(tenant.Identifier),
				Groups:     []iam_model.GroupUUID{group.UUID},
				Roles:      projectBoundRoles,
			}

			err := s.roleBindingRepo.Create(newRoleBinding)
			if err != nil {
				return "", "", err
			}
		}
	}

	serviceAccount, err := s.serviceAccountRepo.GetByIdentifier(tenantUUID, nameForServerRelatedProjectLevelObjects(project.Identifier, serverID))
	if err != nil && !errors.Is(err, iam_model.ErrNotFound) {
		return "", "", err
	}

	if serviceAccount == nil {
		saIdentifier := nameForServerRelatedProjectLevelObjects(project.Identifier, serverID)

		newServiceAccount := &iam_model.ServiceAccount{
			UUID:           uuid.New(),
			Version:        iam_repo.NewResourceVersion(),
			TenantUUID:     tenant.ObjId(),
			Origin:         iam_model.OriginServerAccess,
			Identifier:     saIdentifier,
			FullIdentifier: iam_repo.CalcServiceAccountFullIdentifier(saIdentifier, tenant.Identifier),
		}

		err := s.serviceAccountRepo.Create(newServiceAccount)
		if err != nil {
			return "", "", err
		}

		serviceAccount = newServiceAccount
	}

	var isSAInGroup bool
	for _, saInGroup := range group.ServiceAccounts {
		if saInGroup == serviceAccount.UUID {
			isSAInGroup = true
		}
	}

	if !isSAInGroup {
		group.ServiceAccounts = append(group.ServiceAccounts, serviceAccount.UUID)
	}

	groupService := usecase.Groups(s.tx, tenantUUID)

	if errors.Is(getGroupErr, iam_model.ErrNotFound) {
		err := groupService.Create(group)
		if err != nil {
			return "", "", err
		}
	} else {
		err = groupService.Update(group)
		if err != nil {
			return "", "", err
		}
	}

	var multipassRoleNames []iam_model.RoleName
	for _, tenantRole := range tenantBoundRoles {
		multipassRoleNames = append(multipassRoleNames, tenantRole.Name)
	}
	for _, projectRole := range projectBoundRoles {
		multipassRoleNames = append(multipassRoleNames, projectRole.Name)
	}

	multipassService := usecase.Multipasses(s.tx, iam_model.OriginServerAccess, iam_model.MultipassOwnerServiceAccount, tenantUUID, serviceAccount.UUID)

	// TODO: are these valid?
	multipassJWT, mp, err := multipassService.CreateWithJWT(multipassIssue, 144*time.Hour, 2000*time.Hour, nil, nil, "TODO")
	if err != nil {
		return "", "", err
	}

	server.Version = iam_repo.NewResourceVersion()
	server.MultipassUUID = mp.UUID
	err = s.tx.Insert(model.ServerType, server)
	if err != nil {
		return "", "", err
	}

	return server.UUID, multipassJWT, nil
}

func (s *ServerService) Update(server *model.Server) error {
	stored, err := s.serverRepo.GetByUUID(server.UUID)
	if err != nil {
		return err
	}

	if stored.TenantUUID != server.TenantUUID {
		return iam_model.ErrNotFound
	}
	server.Version = iam_repo.NewResourceVersion()

	project, err := s.projectsService.GetByID(server.ProjectUUID)
	if err != nil {
		return err
	}

	sa, err := s.serviceAccountRepo.GetByIdentifier(server.TenantUUID, nameForServerRelatedProjectLevelObjects(project.Identifier, stored.Identifier))
	if err != nil {
		return err
	}

	sa.Identifier = nameForServerRelatedProjectLevelObjects(project.Identifier, stored.Identifier)
	sa.Version = iam_repo.NewResourceVersion()

	err = s.serviceAccountRepo.Update(sa)
	if err != nil {
		return err
	}

	err = s.serverRepo.Update(server)
	if err != nil {
		return err
	}

	return nil
}

func (s *ServerService) Delete(serverUUID string) error {
	archivingTime := time.Now().Unix()
	archivingHash := rand.Int63n(archivingTime)

	server, err := s.serverRepo.GetByUUID(serverUUID)
	if err != nil {
		return err
	}

	tenant, err := s.tenantService.GetByID(server.TenantUUID)
	if err != nil {
		return err
	}

	project, err := s.projectsService.GetByID(server.ProjectUUID)
	if err != nil {
		return err
	}

	multipassService := usecase.ServiceAccountMultipasses(s.tx, iam_model.OriginServerAccess, tenant.UUID, server.UUID)

	mp, err := multipassService.GetByID(server.MultipassUUID)
	if err != nil {
		return err
	}

	err = multipassService.CascadeDelete(mp.UUID, archivingTime, archivingHash)
	if err != nil {
		return err
	}

	var (
		serversPresentInTenant  bool
		serversPresentInProject bool
	)

	sa, err := s.serviceAccountRepo.GetByIdentifier(tenant.UUID, nameForServerRelatedProjectLevelObjects(project.Identifier, server.Identifier))
	if err != nil {
		return err
	}

	err = s.serviceAccountRepo.CleanChildrenSliceIndexes(sa.UUID)
	if err != nil {
		return err
	}

	err = s.serviceAccountRepo.CascadeDelete(sa.UUID, archivingTime, archivingHash)
	if err != nil {
		return err
	}

	serverList, err := s.serverRepo.List(tenant.UUID, "")
	if err != nil {
		return err
	}

	for _, server := range serverList {
		if serversPresentInTenant && serversPresentInProject {
			break
		}

		if server.TenantUUID == tenant.UUID {
			serversPresentInTenant = true
		}

		if server.ProjectUUID == project.UUID {
			serversPresentInProject = true
		}
	}

	if !serversPresentInProject {
		groupToDelete, err := s.groupRepo.GetByIdentifier(tenant.UUID, nameForTenantLevelObjects(tenant.Identifier))
		if err != nil && !errors.Is(err, iam_model.ErrNotFound) {
			return err
		}

		if groupToDelete != nil {
			err := s.groupRepo.Delete(groupToDelete.UUID, archivingTime, archivingHash)
			if err != nil {
				return err
			}
		}

		// TODO: role scopes
		rbsInProject, err := s.roleBindingRepo.List(tenant.UUID, false)
		if err != nil {
			return err
		}
		for _, rb := range rbsInProject {
			if rb.Origin == iam_model.OriginServerAccess {
				err := s.roleBindingRepo.CascadeDelete(rb.UUID, archivingTime, archivingHash)
				if err != nil {
					return err
				}
			}
		}
	}

	if !serversPresentInTenant {
		// TODO: role scopes
		rbsInProject, err := s.roleBindingRepo.List(tenant.UUID, false)
		if err != nil {
			return err
		}
		for _, rb := range rbsInProject {
			if rb.Origin == iam_model.OriginServerAccess {
				err := s.roleBindingRepo.CascadeDelete(rb.UUID, archivingTime, archivingHash)
				if err != nil {
					return err
				}
			}
		}
	}

	return s.serverRepo.Delete(server.UUID)
}

func nameForTenantLevelObjects(tenantID string) string {
	return "servers/" + tenantID
}

func nameForServerRelatedProjectLevelObjects(projectID, serverID string) string {
	return "servers/" + projectID + "/" + serverID
}
