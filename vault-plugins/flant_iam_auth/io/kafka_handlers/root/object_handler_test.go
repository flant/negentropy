package root

import (
	"crypto"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/utils/tests"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

func getHandler(t *testing.T, storage *io.MemoryStore, msg *io.MsgDecoded) (*io.MemoryStore, func(t *testing.T)) {
	if storage == nil {
		storage = tests.CreateTestStorage(t)
	}

	handle := func(t *testing.T) {
		txn := storage.Txn(true)
		defer txn.Abort()

		objectHandler := NewObjectHandler(hclog.NewNullLogger())
		err := HandleNewMessageIamRootSource(txn, objectHandler, *msg)
		require.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)
	}

	return storage, handle
}

func getCreateUpdateHandler(t *testing.T, obj io.MemoryStorableObject, storage *io.MemoryStore) (*io.MemoryStore, func(t *testing.T)) {
	msg := tests.CreateDecryptCreateMessage(t, obj)
	return getHandler(t, storage, msg)
}

func getDeleteHandler(t *testing.T, obj io.MemoryStorableObject, storage *io.MemoryStore) (*io.MemoryStore, func(t *testing.T)) {
	msg := tests.CreateDecryptDeleteMessage(obj)
	return getHandler(t, storage, msg)
}

func assertCreatedEntityWithAliases(t *testing.T, store *io.MemoryStore, sources []sourceForTest, obj io.MemoryStorableObject, subjectUUID, subjectFullIdentifier string) {
	tx := store.Txn(false)
	defer tx.Abort()

	e, err := repo.NewEntityRepo(tx).GetByUserId(subjectUUID)
	require.NoError(t, err)

	require.NotNil(t, e, "must save entity")
	require.Equal(t, e.UserId, subjectUUID)
	require.Equal(t, e.Name, subjectFullIdentifier, "must name same af full_id")

	_, aliasesBySourceName := getAllAliases(t, tx, subjectUUID)
	for _, s := range sources {
		eaName := s.expectedEaName(obj)
		if eaName != "" {
			require.Contains(t, aliasesBySourceName, s.source.Name, "should create entity alias")
			if aliasesBySourceName[s.source.Name].Name != eaName {
				require.Equal(t, aliasesBySourceName[s.source.Name].Name, eaName, "should correct entity alias name")
			}
		} else {
			require.NotContains(t, aliasesBySourceName, s.source.Name, "should does not create entity alias")
		}
	}
}

func insert(t *testing.T, s *io.MemoryStore, table string, o io.MemoryStorableObject) {
	tx := s.Txn(true)
	defer tx.Abort()
	err := tx.Insert(table, o)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}

func getAllAliases(t *testing.T, tx *io.MemoryStoreTxn, iamModelId string) ([]*model.EntityAlias, map[string]*model.EntityAlias) {
	aliases := make([]*model.EntityAlias, 0)
	aliasesBySourceName := map[string]*model.EntityAlias{}
	err := repo.NewEntityAliasRepo(tx).GetAllForUser(iamModelId, func(a *model.EntityAlias) (bool, error) {
		aliases = append(aliases, a)
		aliasesBySourceName[a.SourceName] = a
		return true, nil
	})
	require.NoError(t, err)

	return aliases, aliasesBySourceName
}

type sourceForTest struct {
	source         *model.AuthSource
	name           string
	expectedEaName func(object io.MemoryStorableObject) string
}

func generateSources(t *testing.T, store *io.MemoryStore) []sourceForTest {
	txn := store.Txn(true)
	defer txn.Abort()

	sources := []sourceForTest{
		{
			name: "email",
			source: &model.AuthSource{
				Name: "s1",

				ParsedJWTPubKeys:     []crypto.PublicKey{"pubkey"},
				JWTValidationPubKeys: []string{"pubkey1"},
				JWTSupportedAlgs:     []string{},
				OIDCResponseTypes:    []string{},
				BoundIssuer:          "http://vault.example.com/",
				NamespaceInState:     true,
				EntityAliasName:      model.EntityAliasNameEmail,
			},

			expectedEaName: func(object io.MemoryStorableObject) string {
				if object.ObjType() == iam_model.UserType {
					return object.(*iam_model.User).Email
				}

				return ""
			},
		},

		{
			name: "full_id",
			source: &model.AuthSource{
				Name: "s2",

				ParsedJWTPubKeys:     []crypto.PublicKey{"pubkey"},
				JWTValidationPubKeys: []string{"pubkey1"},
				JWTSupportedAlgs:     []string{},
				OIDCResponseTypes:    []string{},
				BoundIssuer:          "http://vault.example.com/",
				NamespaceInState:     true,
				EntityAliasName:      model.EntityAliasNameFullIdentifier,
			},

			expectedEaName: func(object io.MemoryStorableObject) string {
				if object.ObjType() == iam_model.UserType {
					return object.(*iam_model.User).FullIdentifier
				}

				return ""
			},
		},

		{
			name: "uuid",
			source: &model.AuthSource{
				Name: "s3",

				ParsedJWTPubKeys:     []crypto.PublicKey{"pubkey"},
				JWTValidationPubKeys: []string{"pubkey1"},
				JWTSupportedAlgs:     []string{},
				OIDCResponseTypes:    []string{},
				BoundIssuer:          "http://vault.example.com/",
				NamespaceInState:     true,
				EntityAliasName:      model.EntityAliasNameUUID,
			},

			expectedEaName: func(object io.MemoryStorableObject) string {
				if object.ObjType() == iam_model.UserType {
					return object.(*iam_model.User).UUID
				}
				return ""
			},
		},

		{
			name: "enable sa uuid",
			source: &model.AuthSource{
				Name: "s4",

				ParsedJWTPubKeys:     []crypto.PublicKey{"pubkey"},
				JWTValidationPubKeys: []string{"pubkey1"},
				JWTSupportedAlgs:     []string{},
				OIDCResponseTypes:    []string{},
				BoundIssuer:          "http://vault.example.com/",
				NamespaceInState:     true,
				AllowServiceAccounts: true,
				EntityAliasName:      model.EntityAliasNameUUID,
			},

			expectedEaName: func(object io.MemoryStorableObject) string {
				switch object.ObjType() {
				case iam_model.UserType:
					return object.(*iam_model.User).UUID
				case iam_model.ServiceAccountType:
					return object.(*iam_model.ServiceAccount).UUID
				}

				return ""
			},
		},

		{
			name: "enable sa full_id",
			source: &model.AuthSource{
				Name: "s5",

				ParsedJWTPubKeys:     []crypto.PublicKey{"pubkey"},
				JWTValidationPubKeys: []string{"pubkey1"},
				JWTSupportedAlgs:     []string{},
				OIDCResponseTypes:    []string{},
				BoundIssuer:          "http://vault.example.com/",
				NamespaceInState:     true,
				AllowServiceAccounts: true,
				EntityAliasName:      model.EntityAliasNameFullIdentifier,
			},

			expectedEaName: func(object io.MemoryStorableObject) string {
				switch object.ObjType() {
				case iam_model.UserType:
					return object.(*iam_model.User).FullIdentifier
				case iam_model.ServiceAccountType:
					return object.(*iam_model.ServiceAccount).FullIdentifier
				}

				return ""
			},
		},

		{
			name: "enable sa email",
			source: &model.AuthSource{
				Name: "s6",

				ParsedJWTPubKeys:     []crypto.PublicKey{"pubkey"},
				JWTValidationPubKeys: []string{"pubkey1"},
				JWTSupportedAlgs:     []string{},
				OIDCResponseTypes:    []string{},
				BoundIssuer:          "http://vault.example.com/",
				NamespaceInState:     true,
				AllowServiceAccounts: true,
				EntityAliasName:      model.EntityAliasNameEmail,
			},

			expectedEaName: func(object io.MemoryStorableObject) string {
				if object.ObjType() == iam_model.UserType {
					return object.(*iam_model.User).Email
				}
				return ""
			},
		},
	}

	repo := repo.NewAuthSourceRepo(txn)
	for _, s := range sources {
		err := repo.Put(s.source)
		require.NoError(t, err)
	}

	err := txn.Commit()
	require.NoError(t, err)

	return sources
}

func TestRootMessageDispatcherCreate(t *testing.T) {
	onlySaveCases := []struct {
		title string
		obj   io.MemoryStorableObject
		get   func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error)
	}{
		{
			title: "tenant",
			obj: &iam_model.Tenant{
				UUID:       utils.UUID(),
				Version:    "1",
				Identifier: "tenant_id",
				FeatureFlags: []iam_model.TenantFeatureFlag{
					{EnabledForNewProjects: true},
				},
			},
			get: func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error) {
				return iam_repo.NewTenantRepository(tx).GetByID(id)
			},
		},
		{
			title: "project",
			obj: &iam_model.Project{
				UUID:       utils.UUID(),
				TenantUUID: utils.UUID(),
				Version:    "1",
				Identifier: "project_id",
			},
			get: func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error) {
				return iam_repo.NewProjectRepository(tx).GetByID(id)
			},
		},
	}

	for _, c := range onlySaveCases {
		t.Run(fmt.Sprintf("saves only iam '%s' model", c.title), func(t *testing.T) {
			store, handle := getCreateUpdateHandler(t, c.obj, nil)
			handle(t)

			txn := store.Txn(false)
			o, err := c.get(txn, c.obj.ObjId())
			require.NoError(t, err)

			tests.AssertDeepEqual(t, c.obj, o)
		})
	}

	cases := []struct {
		title           string
		obj             io.MemoryStorableObject
		get             func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error)
		fullIdProvider  func(io.MemoryStorableObject) string // provide full identifier of serviceAccount or user
		uuidProvider    func(io.MemoryStorableObject) string // provide uuid of serviceaccount or user
		amountOfAliases int
	}{
		{
			title: "user",
			obj: &iam_model.User{
				UUID:           utils.UUID(),
				TenantUUID:     utils.UUID(),
				Version:        "1",
				Identifier:     "user_id",
				FullIdentifier: "user_id@tenant",
				Email:          "user_id@example.com",
				Origin:         consts.OriginIAM,
			},
			get: func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error) {
				return iam_repo.NewUserRepository(tx).GetByID(id)
			},
			fullIdProvider: func(object io.MemoryStorableObject) string {
				return object.(*iam_model.User).FullIdentifier
			},
			uuidProvider: func(object io.MemoryStorableObject) string {
				return object.(*iam_model.User).UUID
			},
			amountOfAliases: 1,
		},

		{
			title: "service account",
			obj: &iam_model.ServiceAccount{
				UUID:           utils.UUID(),
				TenantUUID:     utils.UUID(),
				Version:        "1",
				Identifier:     "user_id",
				FullIdentifier: "user_id@tenant",
				Origin:         consts.OriginIAM,
				CIDRs:          []string{"127.0.0.1/8"},
				TokenTTL:       3 * time.Second,
				TokenMaxTTL:    5 * time.Second,
			},
			get: func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error) {
				return iam_repo.NewServiceAccountRepository(tx).GetByID(id)
			},
			fullIdProvider: func(object io.MemoryStorableObject) string {
				return object.(*iam_model.ServiceAccount).FullIdentifier
			},
			uuidProvider: func(object io.MemoryStorableObject) string {
				return object.(*iam_model.ServiceAccount).UUID
			},
			amountOfAliases: 2,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s with creates entity and entity aliases", c.title), func(t *testing.T) {
			user := c.obj
			uuid := c.uuidProvider(user)

			store, handler := getCreateUpdateHandler(t, user, nil)
			handler(t)

			tx := store.Txn(false)
			u, err := c.get(tx, uuid)
			require.NoError(t, err)
			require.NotNil(t, "must save user in db")
			tests.AssertDeepEqual(t, user, u)

			e, err := repo.NewEntityRepo(tx).GetByUserId(uuid)
			require.NoError(t, err)

			require.NotNil(t, e, "must save entity")
			require.Equal(t, e.UserId, uuid)
			require.Equal(t, e.Name, c.fullIdProvider(user), "must name same af full_id")

			aliases, _ := getAllAliases(t, tx, uuid)

			require.Len(t, aliases, c.amountOfAliases, "should create one alias for user (internal source multipass) and two for sa (multipass and sapassword)")

			t.Run("creates entity aliases for all auth sources", func(t *testing.T) {
				user := c.obj
				uuid := c.uuidProvider(user)

				store, handler := getCreateUpdateHandler(t, user, nil)
				sources := generateSources(t, store)

				handler(t)
				tx := store.Txn(false)
				u, err := c.get(tx, uuid)
				require.NoError(t, err)
				require.NotNil(t, "must save user in db")
				tests.AssertDeepEqual(t, user, u)

				assertCreatedEntityWithAliases(t, store, sources, user, uuid, c.fullIdProvider(user))
			})
		})
	}
}

func TestRootMessageDispatcherDelete(t *testing.T) {
	onlySaveCases := []struct {
		title     string
		obj       io.MemoryStorableObject
		objStale  io.MemoryStorableObject
		get       func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error)
		tableName string
	}{
		{
			title: "tenant",
			obj: &iam_model.Tenant{
				UUID:       utils.UUID(),
				Version:    "1",
				Identifier: "tenant_id",
				FeatureFlags: []iam_model.TenantFeatureFlag{
					{EnabledForNewProjects: true},
				},
			},
			objStale: &iam_model.Tenant{
				UUID:       utils.UUID(),
				Version:    "1",
				Identifier: "tenant_saved",
				FeatureFlags: []iam_model.TenantFeatureFlag{
					{EnabledForNewProjects: true},
				},
			},
			get: func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error) {
				return iam_repo.NewTenantRepository(tx).GetByID(id)
			},
			tableName: iam_model.TenantType,
		},

		{
			title: "project",
			obj: &iam_model.Project{
				UUID:       utils.UUID(),
				TenantUUID: utils.UUID(),
				Version:    "1",
				Identifier: "project_id",
			},

			objStale: &iam_model.Project{
				UUID:       utils.UUID(),
				TenantUUID: utils.UUID(),
				Version:    "1",
				Identifier: "project_staled",
			},

			get: func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error) {
				return iam_repo.NewProjectRepository(tx).GetByID(id)
			},

			tableName: iam_model.ProjectType,
		},
	}

	for _, c := range onlySaveCases {
		t.Run(fmt.Sprintf("deletes only iam '%s' model", c.title), func(t *testing.T) {
			store, handle := getDeleteHandler(t, c.obj, nil)

			insert(t, store, c.tableName, c.obj)
			insert(t, store, c.tableName, c.objStale)

			handle(t)

			txn := store.Txn(false)
			o, err := c.get(txn, c.obj.ObjId())
			require.ErrorIs(t, err, consts.ErrNotFound, "should delete iam entity")

			o, err = c.get(txn, c.objStale.ObjId())
			require.NoError(t, err)
			require.NotNil(t, o, "should delete only one iam entity")
		})
	}

	cases := []struct {
		title     string
		obj       io.MemoryStorableObject
		objStale  io.MemoryStorableObject
		get       func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error)
		fullId    func(io.MemoryStorableObject) string
		id        func(io.MemoryStorableObject) string
		tableName string
	}{
		{
			title: "user",
			obj: &iam_model.User{
				UUID:           utils.UUID(),
				TenantUUID:     utils.UUID(),
				Version:        "1",
				Identifier:     "user_id",
				FullIdentifier: "user_id@tenant",
				Email:          "user_id@example.com",
				Origin:         consts.OriginIAM,
			},
			objStale: &iam_model.User{
				UUID:           utils.UUID(),
				TenantUUID:     utils.UUID(),
				Version:        "1",
				Identifier:     "user_stale",
				FullIdentifier: "user_stale@tenant",
				Email:          "user_stale@example.com",
				Origin:         consts.OriginIAM,
			},

			get: func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error) {
				return iam_repo.NewUserRepository(tx).GetByID(id)
			},
			fullId: func(object io.MemoryStorableObject) string {
				return object.(*iam_model.User).FullIdentifier
			},
			id: func(object io.MemoryStorableObject) string {
				return object.(*iam_model.User).UUID
			},

			tableName: iam_model.UserType,
		},

		{
			title: "service account",
			obj: &iam_model.ServiceAccount{
				UUID:           utils.UUID(),
				TenantUUID:     utils.UUID(),
				Version:        "1",
				Identifier:     "Sa",
				FullIdentifier: "user_id@tenant",
				Origin:         consts.OriginIAM,
				CIDRs:          []string{"127.0.0.1/8"},
				TokenTTL:       3 * time.Second,
				TokenMaxTTL:    5 * time.Second,
			},

			objStale: &iam_model.ServiceAccount{
				UUID:           utils.UUID(),
				TenantUUID:     utils.UUID(),
				Version:        "1",
				Identifier:     "sa_stale",
				FullIdentifier: "sa_stale@tenant",
				Origin:         consts.OriginIAM,
				CIDRs:          []string{"127.0.0.1/8"},
				TokenTTL:       3 * time.Second,
				TokenMaxTTL:    5 * time.Second,
			},

			get: func(tx *io.MemoryStoreTxn, id string) (io.MemoryStorableObject, error) {
				return iam_repo.NewServiceAccountRepository(tx).GetByID(id)
			},
			fullId: func(object io.MemoryStorableObject) string {
				return object.(*iam_model.ServiceAccount).FullIdentifier
			},
			id: func(object io.MemoryStorableObject) string {
				return object.(*iam_model.ServiceAccount).UUID
			},

			tableName: iam_model.ServiceAccountType,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("delete %s delete entity and entity aliases", c.title), func(t *testing.T) {
			objUUID := c.id(c.obj)
			store, handlerCreateObj := getCreateUpdateHandler(t, c.obj, nil)
			sources := generateSources(t, store)
			store, handlerCreateStaleObj := getCreateUpdateHandler(t, c.objStale, store)
			handlerCreateObj(t)
			handlerCreateStaleObj(t)

			// check that all created
			assertCreatedEntityWithAliases(t, store, sources, c.obj, objUUID, c.fullId(c.obj))
			assertCreatedEntityWithAliases(t, store, sources, c.objStale, c.id(c.objStale), c.fullId(c.objStale))

			store, handlerDeleteObj := getDeleteHandler(t, c.obj, store)

			handlerDeleteObj(t)

			tx := store.Txn(false)
			ie, err := c.get(tx, objUUID)
			require.ErrorIs(t, err, consts.ErrNotFound)
			require.Nil(t, ie)

			e, err := repo.NewEntityRepo(tx).GetByUserId(objUUID)
			require.NoError(t, err)
			require.Nil(t, e, "entity must be deleted")

			aliases, _ := getAllAliases(t, tx, objUUID)
			require.Len(t, aliases, 0, "should delete all aliases for iam entity")

			// assert not delete stale object (delete all for only iam entity)
			assertCreatedEntityWithAliases(t, store, sources, c.objStale, c.id(c.objStale), c.fullId(c.objStale))
		})
	}
}
