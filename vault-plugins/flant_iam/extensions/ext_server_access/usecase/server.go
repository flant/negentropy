package usecase

import (
	"errors"
	"fmt"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
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
		tenantService:      usecase.Tenants(tx, consts.OriginServerAccess),
		projectsService:    usecase.Projects(tx, consts.OriginServerAccess),
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
	var projectBoundRoles []iam_model.BoundRole

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

	groupIdentifier := nameForProjectLevelObjects(tenant.Identifier, project.Identifier)
	group, getGroupErr := s.groupRepo.GetByIdentifierAtTenant(tenantUUID, groupIdentifier)
	if getGroupErr != nil && !errors.Is(getGroupErr, consts.ErrNotFound) {
		return "", "", err
	}
	if group == nil {
		group = &iam_model.Group{
			UUID:       uuid.New(),
			TenantUUID: tenant.UUID,
			Identifier: groupIdentifier,
			Origin:     consts.OriginServerAccess,
		}
		err := s.groupRepo.Create(group)
		if err != nil {
			return "", "", err
		}
	}

	// create RoleBinding for each role
	for _, roleName := range roles {
		role, err := s.roleService.Get(roleName)
		if err != nil {
			return "", "", fmt.Errorf("roleService.Get(%q):%w", roleName, err)
		}

		projectBoundRoles = append(projectBoundRoles, iam_model.BoundRole{
			Name: role.Name,
		})
	}

	if len(projectBoundRoles) != 0 {
		var (
			roleBinding *iam_model.RoleBinding
			err         error
		)

		// TODO: update existing
		roleBinding, err = s.roleBindingRepo.FindSpecificRoleBindingAtProject(projectUUID, roles, []string{group.UUID})
		if err != nil && !errors.Is(err, consts.ErrNotFound) {
			return "", "", err
		}

		if roleBinding == nil {
			newRoleBinding := &iam_model.RoleBinding{
				UUID:        uuid.New(),
				TenantUUID:  tenant.ObjId(),
				Version:     iam_repo.NewResourceVersion(),
				Origin:      consts.OriginServerAccess,
				Description: groupIdentifier,
				Groups:      []iam_model.GroupUUID{group.UUID},
				Members: []iam_model.MemberNotation{{
					Type: iam_model.GroupType,
					UUID: group.UUID,
				}},
				Roles:      projectBoundRoles,
				AnyProject: false,
				Projects:   []string{projectUUID},
			}

			err := s.roleBindingRepo.Create(newRoleBinding)
			if err != nil {
				return "", "", err
			}
		}
	}

	serviceAccount, err := s.serviceAccountRepo.GetByIdentifier(tenantUUID, nameForServerRelatedProjectLevelObjects(project.Identifier, serverID))
	if err != nil && !errors.Is(err, consts.ErrNotFound) {
		return "", "", err
	}

	if serviceAccount == nil {
		saIdentifier := nameForServerRelatedProjectLevelObjects(project.Identifier, serverID)

		newServiceAccount := &iam_model.ServiceAccount{
			UUID:           uuid.New(),
			Version:        iam_repo.NewResourceVersion(),
			TenantUUID:     tenant.ObjId(),
			Origin:         consts.OriginServerAccess,
			Identifier:     saIdentifier,
			FullIdentifier: iam_repo.CalcServiceAccountFullIdentifier(saIdentifier, tenant.Identifier),
		}

		err := s.serviceAccountRepo.Create(newServiceAccount)
		if err != nil {
			return "", "", err
		}

		serviceAccount = newServiceAccount
	}

	server.ServiceAccountUUID = serviceAccount.UUID

	var isSAInGroup bool
	for _, saInGroup := range group.ServiceAccounts {
		if saInGroup == serviceAccount.UUID {
			isSAInGroup = true
		}
	}

	if !isSAInGroup {
		group.ServiceAccounts = append(group.ServiceAccounts, serviceAccount.UUID)
		group.Members = append(group.Members, iam_model.MemberNotation{
			Type: iam_model.ServiceAccountType,
			UUID: serviceAccount.UUID,
		})
	}

	groupService := usecase.Groups(s.tx, tenantUUID, consts.OriginServerAccess)

	err = groupService.Update(group)
	if err != nil {
		return "", "", err
	}

	var multipassRoleNames []iam_model.RoleName
	for _, projectRole := range projectBoundRoles {
		multipassRoleNames = append(multipassRoleNames, projectRole.Name)
	}

	multipassService := usecase.Multipasses(s.tx, consts.OriginServerAccess, iam_model.MultipassOwnerServiceAccount, tenantUUID, serviceAccount.UUID)

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
	if stored.Version != server.Version {
		return consts.ErrBadVersion
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	if stored.TenantUUID != server.TenantUUID {
		return consts.ErrNotFound
	}
	server.Version = iam_repo.NewResourceVersion()
	server.MultipassUUID = stored.MultipassUUID
	server.ServiceAccountUUID = stored.ServiceAccountUUID
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

	multipassService := usecase.ServiceAccountMultipasses(s.tx, consts.OriginServerAccess, tenant.UUID, server.ServiceAccountUUID)

	mp, err := multipassService.GetByID(server.MultipassUUID)
	if err != nil {
		return err
	}

	archiveMark := memdb.NewArchiveMark()

	err = multipassService.CascadeDelete(mp.UUID, archiveMark)
	if err != nil {
		return err
	}

	var (
		serversPresentInTenant  bool
		serversPresentInProject bool
	)

	sa, err := s.serviceAccountRepo.GetByID(server.ServiceAccountUUID)
	if err != nil {
		return err
	}

	err = s.serviceAccountRepo.CleanChildrenSliceIndexes(sa.UUID)
	if err != nil {
		return err
	}

	err = s.serviceAccountRepo.CascadeDelete(sa.UUID, archiveMark)
	if err != nil {
		return err
	}

	serverList, err := s.serverRepo.List(tenant.UUID, "", false)
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
		groupToDelete, err := s.groupRepo.GetByIdentifierAtTenant(tenant.UUID, nameForProjectLevelObjects(tenant.Identifier, project.Identifier))
		if err != nil && !errors.Is(err, consts.ErrNotFound) {
			return err
		}

		if groupToDelete != nil {
			err := s.groupRepo.CascadeDelete(groupToDelete.UUID, archiveMark)
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
			if rb.Origin == consts.OriginServerAccess {
				err := s.roleBindingRepo.CascadeDelete(rb.UUID, archiveMark)
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
			if rb.Origin == consts.OriginServerAccess {
				err := s.roleBindingRepo.CascadeDelete(rb.UUID, archiveMark)
				if err != nil {
					return err
				}
			}
		}
	}
	return s.serverRepo.Delete(server.UUID, archiveMark)
}

func nameForProjectLevelObjects(tenantID string, projectID string) string {
	return "servers/" + projectID + "/" + tenantID
}

func nameForServerRelatedProjectLevelObjects(projectID, serverID string) string {
	return "servers/" + projectID + "/" + serverID
}
