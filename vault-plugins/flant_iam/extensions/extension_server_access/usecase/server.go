package usecase

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
)

type ServerService struct {
	tenantService      *usecase.TenantService
	projectsService    *usecase.ProjectService
	groupRepo          *model.GroupRepository
	serviceAccountRepo *model.ServiceAccountRepository
	roleService        *usecase.RoleService
	roleBindingRepo    *model.RoleBindingRepository
	serverRepo         *model2.ServerRepository

	tx *io.MemoryStoreTxn
}

func NewServerService(tx *io.MemoryStoreTxn) *ServerService {
	return &ServerService{
		tenantService:      usecase.Tenants(tx),
		projectsService:    usecase.Projects(tx),
		groupRepo:          model.NewGroupRepository(tx),
		serviceAccountRepo: model.NewServiceAccountRepository(tx),
		roleBindingRepo:    model.NewRoleBindingRepository(tx),
		serverRepo:         model2.NewServerRepository(tx),
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
		tenantBoundRoles  []model.BoundRole
		projectBoundRoles []model.BoundRole
	)

	server := &model2.Server{
		UUID:        uuid.New(),
		TenantUUID:  tenantUUID,
		ProjectUUID: projectUUID,
		Version:     model.NewResourceVersion(),
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
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return "", "", err
	}
	if rawServer != nil {
		return "", "", fmt.Errorf("server with identifier %q already exists in project %q", serverID, project.Identifier)
	}

	group, getGroupErr := s.groupRepo.GetByIdentifier(tenantUUID, nameForTenantLevelObjects(tenant.Identifier))
	if getGroupErr != nil && !errors.Is(getGroupErr, model.ErrNotFound) {
		return "", "", err
	}
	if group == nil {
		group = &model.Group{
			UUID:       uuid.New(),
			TenantUUID: tenant.UUID,
			Identifier: nameForTenantLevelObjects(tenant.Identifier),
			Origin:     model.OriginServerAccess,
		}
	}

	// create RoleBinding for each role
	for _, roleName := range roles {
		role, err := s.roleService.Get(roleName)
		if err != nil {
			return "", "", err
		}

		switch role.Scope {
		case model.RoleScopeTenant:
			tenantBoundRoles = append(tenantBoundRoles, model.BoundRole{
				Name: role.Name,
			})
		case model.RoleScopeProject:
			projectBoundRoles = append(projectBoundRoles, model.BoundRole{
				Name: role.Name,
			})
		}
	}

	// FIXME: remove duplication
	if len(tenantBoundRoles) != 0 {
		var (
			roleBinding *model.RoleBinding
			err         error
		)

		// TODO: update existing
		roleBinding, err = s.roleBindingRepo.GetByIdentifier(tenantUUID, nameForTenantLevelObjects(tenant.Identifier))
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return "", "", err
		}

		if roleBinding == nil {
			newRoleBinding := &model.RoleBinding{
				UUID:       uuid.New(),
				Version:    model.NewResourceVersion(),
				TenantUUID: tenant.ObjId(),
				Origin:     model.OriginServerAccess,
				Identifier: nameForTenantLevelObjects(tenant.Identifier),
				Groups:     []model.GroupUUID{group.UUID},
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
			roleBinding *model.RoleBinding
			err         error
		)

		// TODO: update existing
		roleBinding, err = s.roleBindingRepo.GetByIdentifier(tenantUUID, nameForTenantLevelObjects(tenant.Identifier))
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return "", "", err
		}

		if roleBinding == nil {
			newRoleBinding := &model.RoleBinding{
				UUID:       uuid.New(),
				TenantUUID: tenant.ObjId(),
				Version:    model.NewResourceVersion(),
				Origin:     model.OriginServerAccess,
				Identifier: nameForTenantLevelObjects(tenant.Identifier),
				Groups:     []model.GroupUUID{group.UUID},
				Roles:      projectBoundRoles,
			}

			err := s.roleBindingRepo.Create(newRoleBinding)
			if err != nil {
				return "", "", err
			}
		}
	}

	serviceAccount, err := s.serviceAccountRepo.GetByIdentifier(tenantUUID, nameForServerRelatedProjectLevelObjects(project.Identifier, serverID))
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return "", "", err
	}

	if serviceAccount == nil {
		saIdentifier := nameForServerRelatedProjectLevelObjects(project.Identifier, serverID)

		newServiceAccount := &model.ServiceAccount{
			UUID:           uuid.New(),
			Version:        model.NewResourceVersion(),
			TenantUUID:     tenant.ObjId(),
			Origin:         model.OriginServerAccess,
			Identifier:     saIdentifier,
			FullIdentifier: model.CalcServiceAccountFullIdentifier(saIdentifier, tenant.Identifier),
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

	if errors.Is(getGroupErr, model.ErrNotFound) {
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

	var multipassRoleNames []model.RoleName
	for _, tenantRole := range tenantBoundRoles {
		multipassRoleNames = append(multipassRoleNames, tenantRole.Name)
	}
	for _, projectRole := range projectBoundRoles {
		multipassRoleNames = append(multipassRoleNames, projectRole.Name)
	}

	multipassService := usecase.Multipasses(s.tx, model.OriginServerAccess, model.MultipassOwnerServiceAccount, tenantUUID, serviceAccount.UUID)

	// TODO: are these valid?
	multipassJWT, mp, err := multipassService.CreateWithJWT(multipassIssue, 144*time.Hour, 2000*time.Hour, nil, nil, "TODO")
	if err != nil {
		return "", "", err
	}

	server.Version = model.NewResourceVersion()
	server.MultipassUUID = mp.UUID
	err = s.tx.Insert(model2.ServerType, server)
	if err != nil {
		return "", "", err
	}

	return server.UUID, multipassJWT, nil
}

func (s *ServerService) Update(server *model2.Server) error {
	stored, err := s.serverRepo.GetByUUID(server.UUID)
	if err != nil {
		return err
	}

	if stored.TenantUUID != server.TenantUUID {
		return model.ErrNotFound
	}
	server.Version = model.NewResourceVersion()

	project, err := s.projectsService.GetByID(server.ProjectUUID)
	if err != nil {
		return err
	}

	sa, err := s.serviceAccountRepo.GetByIdentifier(server.TenantUUID, nameForServerRelatedProjectLevelObjects(project.Identifier, stored.Identifier))
	if err != nil {
		return err
	}

	sa.Identifier = nameForServerRelatedProjectLevelObjects(project.Identifier, stored.Identifier)
	sa.Version = model.NewResourceVersion()

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

	multipassService := usecase.ServiceAccountMultipasses(s.tx, model.OriginServerAccess, tenant.UUID, server.UUID)

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

	err = s.serviceAccountRepo.Delete(sa.UUID, archivingTime, archivingHash)
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
		if err != nil && !errors.Is(err, model.ErrNotFound) {
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
			if rb.Origin == model.OriginServerAccess {
				err := s.roleBindingRepo.Delete(rb.UUID, archivingTime, archivingHash)
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
			if rb.Origin == model.OriginServerAccess {
				err := s.roleBindingRepo.Delete(rb.UUID, archivingTime, archivingHash)
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
