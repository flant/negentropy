package repo

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"math/big"
	"sort"
	"time"

	hcmemdb "github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/logical"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func ServerSchema() *memdb.DBSchema {
	var serverIdentifierMultiIndexer []hcmemdb.Indexer

	tenantUUIDIndex := &hcmemdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, tenantUUIDIndex)

	projectUUIDIndex := &hcmemdb.StringFieldIndex{
		Field:     "ProjectUUID",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, projectUUIDIndex)

	serverIdentifierIndex := &hcmemdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, serverIdentifierIndex)

	var tenantProjectMultiIndexer []hcmemdb.Indexer
	tenantProjectMultiIndexer = append(tenantProjectMultiIndexer, tenantUUIDIndex)
	tenantProjectMultiIndexer = append(tenantProjectMultiIndexer, projectUUIDIndex)

	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			ext_model.ServerType: {
				Name: ext_model.ServerType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					iam_repo.PK: {
						Name:   iam_repo.PK,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					iam_repo.TenantForeignPK: {
						Name: iam_repo.TenantForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					iam_repo.ProjectForeignPK: {
						Name: iam_repo.ProjectForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "ProjectUUID",
							Lowercase: true,
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &hcmemdb.CompoundIndex{
							Indexes: serverIdentifierMultiIndexer,
						},
					},
					"tenant_project": {
						Name: "tenant_project",
						Indexer: &hcmemdb.CompoundIndex{
							Indexes: tenantProjectMultiIndexer,
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			ext_model.ServerType: {
				{OriginalDataTypeFieldName: "TenantUUID", RelatedDataType: iam_model.TenantType, RelatedDataTypeFieldIndexName: iam_repo.PK},
				{OriginalDataTypeFieldName: "ProjectUUID", RelatedDataType: iam_model.ProjectType, RelatedDataTypeFieldIndexName: iam_repo.PK},
				// {OriginalDataTypeFieldName: "MultipassUUID", RelatedDataType: iam_model.MultipassType, RelatedDataTypeFieldIndexName: iam_repo.PK}, may have not multipass
			},
		},
		CascadeDeletes: map[string][]memdb.Relation{
			iam_model.TenantType:  {{OriginalDataTypeFieldName: "UUID", RelatedDataType: ext_model.ServerType, RelatedDataTypeFieldIndexName: iam_repo.TenantForeignPK}},
			iam_model.ProjectType: {{OriginalDataTypeFieldName: "UUID", RelatedDataType: ext_model.ServerType, RelatedDataTypeFieldIndexName: iam_repo.ProjectForeignPK}},
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

func (r *ServerRepository) Create(server *ext_model.Server) error {
	return r.db.Insert(ext_model.ServerType, server)
}

func (r *ServerRepository) GetByUUID(id string) (*ext_model.Server, error) {
	raw, err := r.db.First(ext_model.ServerType, iam_repo.PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, iam_model.ErrNotFound
	}

	server := raw.(*ext_model.Server)
	return server, nil
}

func (r *ServerRepository) GetByID(tenant_uuid, project_uuid, id string) (*ext_model.Server, error) {
	raw, err := r.db.First(ext_model.ServerType, "identifier", tenant_uuid, project_uuid, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, iam_model.ErrNotFound
	}

	server := raw.(*ext_model.Server)
	return server, nil
}

func (r *ServerRepository) Update(server *ext_model.Server) error {
	_, err := r.GetByUUID(server.UUID)
	if err != nil {
		return err
	}
	return r.db.Insert(ext_model.ServerType, server)
}

func (r *ServerRepository) Delete(uuid string, archivingTimestamp iam_model.UnixTime, archivingHash int64) error {
	server, err := r.GetByUUID(uuid)
	if err != nil {
		return err
	}
	if server.Archived() {
		return iam_model.ErrIsArchived
	}
	return r.db.Archive(ext_model.ServerType, server, archivingTimestamp, archivingHash)
}

func (r *ServerRepository) List(tenantID, projectID string) ([]*ext_model.Server, error) {
	var (
		iter hcmemdb.ResultIterator
		err  error
	)

	switch {
	case tenantID != "" && projectID != "":
		iter, err = r.db.Get(ext_model.ServerType, "tenant_project", tenantID, projectID)

	case tenantID != "":
		iter, err = r.db.Get(ext_model.ServerType, iam_repo.TenantForeignPK, tenantID)

	case projectID != "":
		iter, err = r.db.Get(ext_model.ServerType, iam_repo.ProjectForeignPK, projectID)

	default:
		iter, err = r.db.Get(ext_model.ServerType, iam_repo.PK)
	}
	if err != nil {
		return nil, err
	}

	ids := make([]*ext_model.Server, 0)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*ext_model.Server)
		ids = append(ids, u)
	}
	return ids, nil
}

const uidKey = "uidKey"

type UserServerAccessRepository struct {
	db                              *io.MemoryStoreTxn
	serverRepo                      *ServerRepository
	userRepo                        *iam_repo.UserRepository
	vaultStore                      logical.Storage
	expireSeedAfterRevealIn         time.Duration
	deleteExpiredPasswordSeedsAfter time.Duration
}

func NewUserServerAccessRepository(
	tx *io.MemoryStoreTxn, initialUID int, expireSeedAfterRevealIn, deleteExpiredPasswordSeedsAfter time.Duration,
	vaultStore logical.Storage) (*UserServerAccessRepository, error) {
	r := &UserServerAccessRepository{
		db:                              tx,
		userRepo:                        iam_repo.NewUserRepository(tx),
		serverRepo:                      NewServerRepository(tx),
		vaultStore:                      vaultStore,
		expireSeedAfterRevealIn:         expireSeedAfterRevealIn,
		deleteExpiredPasswordSeedsAfter: deleteExpiredPasswordSeedsAfter,
	}

	err := r.saveUID(initialUID)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *UserServerAccessRepository) saveUID(value int) error {
	return r.vaultStore.Put(context.Background(), &logical.StorageEntry{
		Key:      uidKey,
		Value:    big.NewInt(int64(value)).Bytes(),
		SealWrap: false,
	})
}

func (r *UserServerAccessRepository) getUID() (int, error) {
	entry, err := r.vaultStore.Get(context.Background(), uidKey)
	if err != nil {
		return 0, err
	}
	value := big.NewInt(0).SetBytes(entry.Value).Uint64()
	return int(value), nil
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

	lastUUID, err := r.getUID()
	if err != nil {
		return nil
	}
	currentUUID := lastUUID + 1

	user.Extensions[iam_model.OriginServerAccess] = &iam_model.Extension{
		Origin:    iam_model.OriginServerAccess,
		OwnerType: iam_model.ExtensionOwnerTypeUser,
		OwnerUUID: user.ObjId(),
		Attributes: map[string]interface{}{
			"UID": currentUUID,
			"passwords": []ext_model.UserServerPassword{
				{
					Seed:      randomSeed,
					Salt:      randomSalt,
					ValidTill: time.Time{},
				},
			},
		},
		SensitiveAttributes: nil, // TODO: ?
	}

	return r.saveUID(currentUUID)
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
	passwords := passwordsRaw.([]ext_model.UserServerPassword)

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

func returnFreshPassword(usps []ext_model.UserServerPassword) (ext_model.UserServerPassword, error) {
	if len(usps) == 0 {
		return ext_model.UserServerPassword{}, errors.New("no User password found")
	}

	sort.Slice(usps, func(i, j int) bool {
		return usps[i].ValidTill.Before(usps[j].ValidTill) // TODO: should iterate from freshest. check!!!
	})

	return usps[0], NoValidPasswords
}

func garbageCollectPasswords(usps []ext_model.UserServerPassword, seed, salt []byte,
	expirePasswordSeedAfterRevealIn, deleteAfter time.Duration) (ret []ext_model.UserServerPassword) {
	var (
		currentTime                            = time.Now()
		expirePasswordSeedAfterTimestamp       = currentTime.Add(expirePasswordSeedAfterRevealIn)
		expirePasswordSeedAfterTimestampHalved = currentTime.Add(expirePasswordSeedAfterRevealIn / 2)
		deleteAfterTimestamp                   = currentTime.Add(deleteAfter)
	)

	if !usps[len(usps)-1].ValidTill.After(expirePasswordSeedAfterTimestampHalved) {
		usps[len(usps)-1].ValidTill = time.Time{}
		usps = append(usps, ext_model.UserServerPassword{
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
