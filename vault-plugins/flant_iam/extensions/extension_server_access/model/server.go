package model

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
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
					model.PK: {
						Name:   model.PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					model.TenantForeignPK: {
						Name: model.TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					model.ProjectForeignPK: {
						Name: model.ProjectForeignPK,
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
	UUID          string `json:"uuid"` // ID
	TenantUUID    string `json:"tenant_uuid"`
	ProjectUUID   string `json:"project_uuid"`
	Version       string `json:"resource_version"`
	Identifier    string `json:"identifier"`
	MultipassUUID string `json:"multipass_uuid"`

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
	db *io.MemoryStoreTxn
}

func NewServerRepository(tx *io.MemoryStoreTxn) *ServerRepository {
	return &ServerRepository{
		db: tx,
	}
}

func (r *ServerRepository) Create(server *Server) error {
	return r.db.Insert(ServerType, server)
}

func (r *ServerRepository) GetByUUID(id string) (*Server, error) {
	raw, err := r.db.First(ServerType, model.PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}

	server := raw.(*Server)
	return server, nil
}

func (r *ServerRepository) GetByID(tenant_uuid, project_uuid, id string) (*Server, error) {
	raw, err := r.db.First(ServerType, "identifier", tenant_uuid, project_uuid, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}

	server := raw.(*Server)
	return server, nil
}

func (r *ServerRepository) Update(server *Server) error {
	_, err := r.GetByUUID(server.UUID)
	if err != nil {
		return err
	}
	return r.db.Insert(ServerType, server)
}

func (r *ServerRepository) Delete(uuid string) error {
	server, err := r.GetByUUID(uuid)
	if err != nil {
		return err
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
		iter, err = r.db.Get(ServerType, model.TenantForeignPK, tenantID)

	case projectID != "":
		iter, err = r.db.Get(ServerType, model.ProjectForeignPK, projectID)

	default:
		iter, err = r.db.Get(ServerType, model.PK)
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
	userRepo                        *model.UserRepository
	currentUID                      int // FIXME: commit to Vault local storage
	expireSeedAfterRevealIn         time.Duration
	deleteExpiredPasswordSeedsAfter time.Duration
}

func NewUserServerAccessRepository(
	tx *io.MemoryStoreTxn, initialUID int, expireSeedAfterRevealIn, deleteExpiredPasswordSeedsAfter time.Duration,
) *UserServerAccessRepository {
	return &UserServerAccessRepository{
		db:                              tx,
		userRepo:                        model.NewUserRepository(tx),
		serverRepo:                      NewServerRepository(tx),
		currentUID:                      initialUID,
		expireSeedAfterRevealIn:         expireSeedAfterRevealIn,
		deleteExpiredPasswordSeedsAfter: deleteExpiredPasswordSeedsAfter,
	}
}

func (r *UserServerAccessRepository) CreateExtension(user *model.User) error {
	if user.Extensions == nil {
		user.Extensions = map[model.ObjectOrigin]*model.Extension{}
	}

	if _, ok := user.Extensions[model.OriginServerAccess]; ok {
		return nil
	}

	randomSeed, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return nil
	}

	randomSalt, err := generateRandomBytes(64) // TODO: proper value
	if err != nil {
		return nil
	}

	user.Extensions[model.OriginServerAccess] = &model.Extension{
		Origin:    model.OriginServerAccess,
		OwnerType: model.ExtensionOwnerTypeUser,
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

	passwordsRaw := user.Extensions[model.OriginServerAccess].Attributes["passwords"]
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
