package model

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	ServerType = "server" // also, memdb schema name
)

func ServerSchema() *memdb.DBSchema {
	var serverIdentifierMultiIndexer []memdb.Indexer

	tenantUUIDIndex := &memdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, tenantUUIDIndex)

	projectUUIDIndex := &memdb.StringFieldIndex{
		Field:     "ProjectUUID",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, projectUUIDIndex)

	serverIdentifierIndex := &memdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, serverIdentifierIndex)

	var tenantProjectMultiIndexer []memdb.Indexer
	tenantProjectMultiIndexer = append(tenantProjectMultiIndexer, tenantUUIDIndex)
	tenantProjectMultiIndexer = append(tenantProjectMultiIndexer, projectUUIDIndex)

	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ServerType: {
				Name: ServerType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					ProjectForeignPK: {
						Name: ProjectForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "ProjectUUID",
							Lowercase: true,
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &memdb.CompoundIndex{
							Indexes: serverIdentifierMultiIndexer,
						},
					},
					"tenant_project": {
						Name: "tenant_project",
						Indexer: &memdb.CompoundIndex{
							Indexes: tenantProjectMultiIndexer,
						},
					},
				},
			},
		},
	}
}

type Server struct {
	UUID        string `json:"uuid"` // ID
	TenantUUID  string `json:"tenant_uuid"`
	ProjectUUID string `json:"project_uuid"`
	Version     string `json:"resource_version"`
	Identifier  string `json:"identifier"`

	Fingerprint string            `json:"fingerprint"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	ConnectionInfo ConnectionInfo `json:"connection_info"`
}

type ConnectionInfo struct {
	Hostname     string `json:"hostname"`
	Port         string `json:"port"`
	JumpHostname string `json:"jump_hostname"`
	JumpPort     string `json:"jump_port"`
}

func (c *ConnectionInfo) FillDefaultPorts() {
	if c.Port == "" {
		c.Port = "22"
	}
	if c.JumpHostname != "" && c.JumpPort == "" {
		c.JumpPort = "22"
	}
}

func (u *Server) ObjType() string {
	return ServerType
}

func (u *Server) ObjId() string {
	return u.UUID
}

func (u *Server) AsMap() map[string]interface{} {
	var res map[string]interface{}

	data, _ := json.Marshal(u)

	_ = json.Unmarshal(data, &res)

	return res
}

type ServerRepository struct {
	db                 *io.MemoryStoreTxn
	tenantRepo         *TenantRepository
	projectRepo        *ProjectRepository
	groupRepo          *GroupRepository
	roleRepo           *RoleRepository
	roleBindingRepo    *RoleBindingRepository
	serviceAccountRepo *ServiceAccountRepository
	multipassRepo      *MultipassRepository
}

func NewServerRepository(tx *io.MemoryStoreTxn) *ServerRepository {
	return &ServerRepository{
		db:                 tx,
		tenantRepo:         NewTenantRepository(tx),
		projectRepo:        NewProjectRepository(tx),
		groupRepo:          NewGroupRepository(tx),
		roleRepo:           NewRoleRepository(tx),
		roleBindingRepo:    NewRoleBindingRepository(tx),
		serviceAccountRepo: NewServiceAccountRepository(tx),
		multipassRepo:      NewMultipassRepository(tx),
	}
}

func (r *ServerRepository) Create(server *Server, roles []string) error {
	var (
		tenant         *Tenant
		project        *Project
		group          *Group
		serviceAccount *ServiceAccount

		tenantBoundRoles  []BoundRole
		projectBoundRoles []BoundRole

		err error
	)

	tenant, err = r.tenantRepo.GetByID(server.TenantUUID)
	if err != nil {
		return err
	}

	project, err = r.projectRepo.GetByID(server.ProjectUUID)
	if err != nil {
		return err
	}

	rawServer, err := r.db.First(ServerType, "identifier", tenant.UUID, project.UUID, server.Identifier)
	if err != nil {
		return err
	}
	if rawServer != nil {
		return fmt.Errorf("server with identifier %q already exists in project %q", server.Identifier, project.Identifier)
	}

	group, getGroupErr := r.groupRepo.GetByIDAndTenant(fmt.Sprintf("servers/%s", project.Identifier), tenant.Identifier)
	if getGroupErr != nil && !errors.Is(getGroupErr, ErrNotFound) {
		return err
	}

	if group == nil {
		newGroup := &Group{
			UUID:       uuid.New(),
			TenantUUID: tenant.ObjId(),
			Origin:     OriginServerAccess,
			Identifier: fmt.Sprintf("servers/%s", project.Identifier),
		}
		newGroup.FullIdentifier = CalcGroupFullIdentifier(newGroup.Identifier, tenant.Identifier)

		group = newGroup
	}

	// create RoleBinding for each role
	for _, roleName := range roles {
		role, err := r.roleRepo.Get(roleName)
		if err != nil {
			return err
		}

		switch role.Scope {
		case RoleScopeTenant:
			tenantBoundRoles = append(tenantBoundRoles, BoundRole{
				Name:  role.Name,
				Scope: RoleScopeTenant,
			})
		case RoleScopeProject:
			projectBoundRoles = append(projectBoundRoles, BoundRole{
				Name:  role.Name,
				Scope: RoleScopeProject,
			})
		}
	}

	// FIXME: remove duplication
	if len(tenantBoundRoles) != 0 {
		var (
			roleBinding *RoleBinding
			err         error
		)

		roleBinding, err = r.roleBindingRepo.GetByIdentifier(fmt.Sprintf("servers/%s", server.Identifier), tenant.Identifier)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		if roleBinding == nil {
			newRoleBinding := &RoleBinding{
				UUID:       uuid.New(),
				TenantUUID: tenant.ObjId(),
				Origin:     "server_access", // TODO: ?
				Groups:     []GroupUUID{group.UUID},
				Roles:      tenantBoundRoles,
			}

			err := r.roleBindingRepo.Create(newRoleBinding)
			if err != nil {
				return err
			}
		}
	}

	if len(projectBoundRoles) != 0 {
		var (
			roleBinding *RoleBinding
			err         error
		)

		roleBinding, err = r.roleBindingRepo.GetByIdentifier(fmt.Sprintf("servers/%s", server.Identifier), tenant.Identifier)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		if roleBinding == nil {
			newRoleBinding := &RoleBinding{
				UUID:       uuid.New(),
				TenantUUID: tenant.ObjId(),
				Origin:     OriginServerAccess,
				Groups:     []GroupUUID{group.UUID},
				Roles:      projectBoundRoles,
			}

			err := r.roleBindingRepo.Create(newRoleBinding)
			if err != nil {
				return err
			}
		}
	}

	serviceAccount, err = r.serviceAccountRepo.GetByIdentifier(fmt.Sprintf("server/%s/%s", project.Identifier, server.Identifier), tenant.Identifier)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	if serviceAccount == nil {
		newServiceAccount := &ServiceAccount{
			UUID:       uuid.New(),
			TenantUUID: tenant.ObjId(),
			Origin:     OriginServerAccess,
			Identifier: fmt.Sprintf("server/%s/%s", project.Identifier, server.Identifier),
		}

		err := r.serviceAccountRepo.Create(newServiceAccount)
		if err != nil {
			return err
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

	if errors.Is(getGroupErr, ErrNotFound) {
		err := r.groupRepo.Create(group)
		if err != nil {
			return err
		}
	} else {
		err = r.groupRepo.Update(group)
		if err != nil {
			return err
		}
	}

	server.Version = NewResourceVersion()
	err = r.db.Insert(ServerType, server)
	if err != nil {
		return err
	}

	var multipassRoleNames []RoleName
	for _, tenantRole := range tenantBoundRoles {
		multipassRoleNames = append(multipassRoleNames, tenantRole.Name)
	}
	for _, projectRole := range projectBoundRoles {
		multipassRoleNames = append(multipassRoleNames, projectRole.Name)
	}

	mp := &Multipass{
		UUID:       uuid.New(),
		TenantUUID: tenant.UUID,
		OwnerUUID:  serviceAccount.UUID,
		OwnerType:  ServiceAccountType,
		TTL:        24 * time.Hour, // TODO: change placeholders
		MaxTTL:     72 * time.Hour,
		ValidTill:  time.Now().Add(144 * time.Hour).Unix(),
		Roles:      multipassRoleNames,
		Salt:       "", // TODO: should it be empty?
		Origin:     OriginServerAccess,
	}

	err = r.multipassRepo.Create(mp)
	if err != nil {
		return err
	}

	// TODO: return signed multipass
	return nil
}

func (r *ServerRepository) GetById(id string) (*Server, error) {
	raw, err := r.db.First(ServerType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}

	server := raw.(*Server)
	return server, nil
}

func (r *ServerRepository) Update(server *Server) error {
	stored, err := r.GetById(server.UUID)
	if err != nil {
		return err
	}

	if stored.TenantUUID != server.TenantUUID {
		return ErrNotFound
	}
	server.Version = NewResourceVersion()

	project, err := r.projectRepo.GetByID(server.ProjectUUID)
	if err != nil {
		return err
	}

	sa, err := r.serviceAccountRepo.GetByIdentifier(fmt.Sprintf("server/%s/%s", project.Identifier, stored.Identifier), project.Identifier)
	if err != nil {
		return err
	}

	sa.Identifier = fmt.Sprintf("server/%s/%s", project.Identifier, server.Identifier)

	err = r.serviceAccountRepo.Update(sa)
	if err != nil {
		return err
	}

	err = r.db.Insert(ServerType, server)
	if err != nil {
		return err
	}

	return nil
}

func (r *ServerRepository) Delete(id string) error {
	server, err := r.GetById(id)
	if err != nil {
		return err
	}

	tenant, err := r.tenantRepo.GetByID(server.TenantUUID)
	if err != nil {
		return err
	}

	project, err := r.projectRepo.GetByID(server.ProjectUUID)
	if err != nil {
		return err
	}

	sa, err := r.serviceAccountRepo.GetByIdentifier(fmt.Sprintf("server/%s/%s", project.Identifier, server.Identifier), tenant.UUID)
	if err != nil {
		return err
	}

	err = r.multipassRepo.Delete(&Multipass{
		TenantUUID: tenant.UUID,
		OwnerUUID:  sa.UUID,
		OwnerType:  ServiceAccountType,
	})
	if err != nil {
		return err
	}

	// TODO: delete SA from roles

	err = r.serviceAccountRepo.Delete(OriginServerAccess, sa.UUID)
	if err != nil {
		return err
	}

	var (
		serversPresentInTenant  bool
		serversPresentInProject bool
	)

	serverList, err := r.List(tenant.UUID, "")
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
		groupToDelete, err := r.groupRepo.GetByIDAndTenant(fmt.Sprintf("servers/%s", project.Identifier), tenant.UUID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		if groupToDelete != nil {
			err := r.groupRepo.Delete(OriginServerAccess, groupToDelete.UUID)
			if err != nil {
				return err
			}
		}

		// TODO: role scopes
		rbsInProject, err := r.roleBindingRepo.List(tenant.UUID)
		if err != nil {
			return err
		}
		for _, rb := range rbsInProject {
			if rb.Origin == OriginServerAccess {
				err := r.roleBindingRepo.Delete(OriginServerAccess, rb.UUID)
				if err != nil {
					return err
				}
			}
		}
	}

	if !serversPresentInTenant {
		// TODO: role scopes
		rbsInProject, err := r.roleBindingRepo.List(tenant.UUID)
		if err != nil {
			return err
		}
		for _, rb := range rbsInProject {
			if rb.Origin == OriginServerAccess {
				err := r.roleBindingRepo.Delete(OriginServerAccess, rb.UUID)
				if err != nil {
					return err
				}
			}
		}
	}

	return r.db.Delete(ServerType, server)
}

func (r *ServerRepository) List(tenantID, projectID string) ([]*Server, error) {
	var (
		iter memdb.ResultIterator
		err  error
	)

	switch {
	case tenantID != "" && projectID != "":
		iter, err = r.db.Get(ServerType, "tenant_project", tenantID, projectID)

	case tenantID != "":
		iter, err = r.db.Get(ServerType, TenantForeignPK, tenantID)

	case projectID != "":
		iter, err = r.db.Get(ServerType, ProjectForeignPK, projectID)

	default:
		iter, err = r.db.Get(ServerType, PK)
	}
	if err != nil {
		return nil, err
	}

	ids := make([]*Server, 0)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*Server)
		ids = append(ids, u)
	}
	return ids, nil
}

type UserServerPassword struct {
	Seed      []byte    `json:"seed"`
	Salt      []byte    `json:"salt"`
	ValidTill time.Time `json:"valid_till"`
}

type UserServerAccessRepository struct {
	db                              *io.MemoryStoreTxn
	serverRepo                      *ServerRepository
	userRepo                        *UserRepository
	currentUID                      int // FIXME: commit to Vault local storage
	expireSeedAfterRevealIn         time.Duration
	deleteExpiredPasswordSeedsAfter time.Duration
}

func NewUserServerAccessRepository(
	tx *io.MemoryStoreTxn, initialUID int, expireSeedAfterRevealIn, deleteExpiredPasswordSeedsAfter time.Duration,
) *UserServerAccessRepository {

	return &UserServerAccessRepository{
		db:                              tx,
		userRepo:                        NewUserRepository(tx),
		serverRepo:                      NewServerRepository(tx),
		currentUID:                      initialUID,
		expireSeedAfterRevealIn:         expireSeedAfterRevealIn,
		deleteExpiredPasswordSeedsAfter: deleteExpiredPasswordSeedsAfter,
	}
}

func (r *UserServerAccessRepository) CreateExtension(user *User) error {
	if _, ok := user.Extensions[OriginServerAccess]; ok {
		return nil
	}

	randomSeed, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return err
	}

	randomSalt, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return err
	}

	err = r.userRepo.SetExtension(&Extension{
		Origin:    OriginServerAccess,
		OwnerType: ExtensionOwnerTypeUser,
		OwnerUUID: user.ObjId(),
		Attributes: map[string]interface{}{
			"UID": r.currentUID,
			"passwords": []UserServerPassword{
				{
					Seed:      randomSeed,
					Salt:      randomSalt,
					ValidTill: time.Time{},
				},
			},
		},
		SensitiveAttributes: nil, // TODO: ?
	})
	if err != nil {
		return err
	}

	r.currentUID++

	return nil
}

func (r UserServerAccessRepository) RevealPassword(userUUID, serverUUID string) (string, error) {
	user, err := r.userRepo.GetByID(userUUID)
	if err != nil {
		return "", err
	}

	randomSeed, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return "", err
	}

	randomSalt, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return "", err
	}

	passwordsRaw := user.Extensions[OriginServerAccess].Attributes["passwords"]
	passwords := passwordsRaw.([]UserServerPassword)

	passwords = garbageCollectPasswords(passwords, randomSeed, randomSalt, r.expireSeedAfterRevealIn, r.deleteExpiredPasswordSeedsAfter)

	freshPass, err := returnFreshPassword(passwords)
	if err != nil {
		return "", err
	}

	sha512Hash := sha512.New()
	_, err = sha512Hash.Write(append([]byte(serverUUID), freshPass.Seed...))
	retPass := hex.EncodeToString(sha512Hash.Sum(nil))

	return retPass[:11], nil
}

var NoValidPasswords = errors.New("no valid Password found in User extension")

func returnFreshPassword(usps []UserServerPassword) (UserServerPassword, error) {
	if len(usps) == 0 {
		return UserServerPassword{}, errors.New("no User password found")
	}

	sort.Slice(usps, func(i, j int) bool {
		return usps[i].ValidTill.Before(usps[j].ValidTill) // TODO: should iterate from freshest. check!!!
	})

	return usps[0], NoValidPasswords
}

func garbageCollectPasswords(usps []UserServerPassword, seed, salt []byte,
	expirePasswordSeedAfterRevealIn, deleteAfter time.Duration) (ret []UserServerPassword) {

	var (
		currentTime                            = time.Now()
		expirePasswordSeedAfterTimestamp       = currentTime.Add(expirePasswordSeedAfterRevealIn)
		expirePasswordSeedAfterTimestampHalved = currentTime.Add(expirePasswordSeedAfterRevealIn / 2)
		deleteAfterTimestamp                   = currentTime.Add(deleteAfter)
	)

	if !usps[len(usps)-1].ValidTill.After(expirePasswordSeedAfterTimestampHalved) {
		usps[len(usps)-1].ValidTill = time.Time{}
		usps = append(usps, UserServerPassword{
			Seed:      seed,
			Salt:      salt,
			ValidTill: expirePasswordSeedAfterTimestamp,
		})
	}

	for _, usp := range usps {
		if !usp.ValidTill.Before(deleteAfterTimestamp) {
			ret = append(ret, usp)
		}
	}

	return
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
