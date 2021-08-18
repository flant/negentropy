package repo

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"sort"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
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
			model.ServerType: {
				Name: model.ServerType,
				Indexes: map[string]*memdb.IndexSchema{
					iam_model.PK: {
						Name:   iam_model.PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					iam_model.TenantForeignPK: {
						Name: iam_model.TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					iam_model.ProjectForeignPK: {
						Name: iam_model.ProjectForeignPK,
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

type ServerRepository struct {
	db *io.MemoryStoreTxn
}

func NewServerRepository(tx *io.MemoryStoreTxn) *ServerRepository {
	return &ServerRepository{
		db: tx,
	}
}

func (r *ServerRepository) Create(server *model.Server) error {
	return r.db.Insert(model.ServerType, server)
}

func (r *ServerRepository) GetByUUID(id string) (*model.Server, error) {
	raw, err := r.db.First(model.ServerType, iam_model.PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, iam_model.ErrNotFound
	}

	server := raw.(*model.Server)
	return server, nil
}

func (r *ServerRepository) GetByID(tenant_uuid, project_uuid, id string) (*model.Server, error) {
	raw, err := r.db.First(model.ServerType, "identifier", tenant_uuid, project_uuid, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, iam_model.ErrNotFound
	}

	server := raw.(*model.Server)
	return server, nil
}

func (r *ServerRepository) Update(server *model.Server) error {
	_, err := r.GetByUUID(server.UUID)
	if err != nil {
		return err
	}
	return r.db.Insert(model.ServerType, server)
}

func (r *ServerRepository) Delete(uuid string) error {
	server, err := r.GetByUUID(uuid)
	if err != nil {
		return err
	}
	return r.db.Delete(model.ServerType, server)
}

func (r *ServerRepository) List(tenantID, projectID string) ([]*model.Server, error) {
	var (
		iter memdb.ResultIterator
		err  error
	)

	switch {
	case tenantID != "" && projectID != "":
		iter, err = r.db.Get(model.ServerType, "tenant_project", tenantID, projectID)

	case tenantID != "":
		iter, err = r.db.Get(model.ServerType, iam_model.TenantForeignPK, tenantID)

	case projectID != "":
		iter, err = r.db.Get(model.ServerType, iam_model.ProjectForeignPK, projectID)

	default:
		iter, err = r.db.Get(model.ServerType, iam_model.PK)
	}
	if err != nil {
		return nil, err
	}

	ids := make([]*model.Server, 0)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*model.Server)
		ids = append(ids, u)
	}
	return ids, nil
}

type UserServerAccessRepository struct {
	db                              *io.MemoryStoreTxn
	serverRepo                      *ServerRepository
	userRepo                        *iam_model.UserRepository
	currentUID                      int // FIXME: commit to Vault local storage
	expireSeedAfterRevealIn         time.Duration
	deleteExpiredPasswordSeedsAfter time.Duration
}

func NewUserServerAccessRepository(
	tx *io.MemoryStoreTxn, initialUID int, expireSeedAfterRevealIn, deleteExpiredPasswordSeedsAfter time.Duration,
) *UserServerAccessRepository {
	return &UserServerAccessRepository{
		db:                              tx,
		userRepo:                        iam_model.NewUserRepository(tx),
		serverRepo:                      NewServerRepository(tx),
		currentUID:                      initialUID,
		expireSeedAfterRevealIn:         expireSeedAfterRevealIn,
		deleteExpiredPasswordSeedsAfter: deleteExpiredPasswordSeedsAfter,
	}
}

func (r *UserServerAccessRepository) CreateExtension(user *iam_model.User) error {
	if user.Extensions == nil {
		user.Extensions = map[iam_model.ObjectOrigin]*iam_model.Extension{}
	}

	if _, ok := user.Extensions[iam_model.OriginServerAccess]; ok {
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

	user.Extensions[iam_model.OriginServerAccess] = &iam_model.Extension{
		Origin:    iam_model.OriginServerAccess,
		OwnerType: iam_model.ExtensionOwnerTypeUser,
		OwnerUUID: user.ObjId(),
		Attributes: map[string]interface{}{
			"UID": r.currentUID,
			"passwords": []model.UserServerPassword{
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

	passwordsRaw := user.Extensions[iam_model.OriginServerAccess].Attributes["passwords"]
	passwords := passwordsRaw.([]model.UserServerPassword)

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

func returnFreshPassword(usps []model.UserServerPassword) (model.UserServerPassword, error) {
	if len(usps) == 0 {
		return model.UserServerPassword{}, errors.New("no User password found")
	}

	sort.Slice(usps, func(i, j int) bool {
		return usps[i].ValidTill.Before(usps[j].ValidTill) // TODO: should iterate from freshest. check!!!
	})

	return usps[0], NoValidPasswords
}

func garbageCollectPasswords(usps []model.UserServerPassword, seed, salt []byte,
	expirePasswordSeedAfterRevealIn, deleteAfter time.Duration) (ret []model.UserServerPassword) {
	var (
		currentTime                            = time.Now()
		expirePasswordSeedAfterTimestamp       = currentTime.Add(expirePasswordSeedAfterRevealIn)
		expirePasswordSeedAfterTimestampHalved = currentTime.Add(expirePasswordSeedAfterRevealIn / 2)
		deleteAfterTimestamp                   = currentTime.Add(deleteAfter)
	)

	if !usps[len(usps)-1].ValidTill.After(expirePasswordSeedAfterTimestampHalved) {
		usps[len(usps)-1].ValidTill = time.Time{}
		usps = append(usps, model.UserServerPassword{
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
