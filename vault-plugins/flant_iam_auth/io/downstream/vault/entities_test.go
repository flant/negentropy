package vault

import (
	"context"
	"encoding/hex"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	vault_identity "github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

func getDownStreamApi() (*VaultEntityDownstreamApi, *io.MemoryStore, *api.Client, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, nil, nil, err
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		token = "root"
	}

	client.SetToken(token)

	storageView := &logical.InmemStorage{}

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), storageView)
	if err != nil {
		return nil, nil, nil, err
	}

	schema, err := repo.GetSchema()
	if err != nil {
		return nil, nil, nil, err
	}

	storage, err := io.NewMemoryStore(schema, mb)

	getClient := func() (*api.Client, error) {
		return client, nil
	}

	return &VaultEntityDownstreamApi{
		getClient:           getClient,
		mountAccessorGetter: NewMountAccessorGetter(getClient, "token/"),
		logger:              hclog.NewNullLogger(),
	}, storage, client, nil
}

func skipNoneDev(t *testing.T) {
	if os.Getenv("VAULT_ADDR") == "" {
		t.Skip("vault does not start")
	}
}

func randomStr() string {
	rand.Seed(time.Now().UnixNano())

	entityName := make([]byte, 20)
	_, err := rand.Read(entityName)
	if err != nil {
		panic("not generate entity name")
	}

	return hex.EncodeToString(entityName)
}

func TestEntitiesProcess(t *testing.T) {
	skipNoneDev(t)

	TestEntites_SkipIncorrect(t)
	TestEntites_WriteNewEntity(t)
	TestEntites_WriteNewEntityAlias(t)
	TestEntites_DeleteEntity(t)
	TestEntites_DeleteEntityAlias(t)
}

func TestEntites_SkipIncorrect(t *testing.T) {
	entities := []io.MemoryStorableObject{
		&iam.User{},
		&iam.Group{},
		&model.AuthSource{},
		&model.AuthMethod{},
	}

	for _, e := range entities {
		t.Run("skip not support entities", func(t *testing.T) {
			skipNoneDev(t)

			downstream, storage, _, err := getDownStreamApi()
			if err != nil {
				t.Fatal("does not ger api")
			}

			txn := storage.Txn(true)

			actions, err := downstream.ProcessObject(storage, txn, e)
			if err != nil {
				t.Fatal("raise error", err)
			}

			if len(actions) > 0 {
				t.Fatal("has actions")
			}

			txn.Abort()
		})
	}
}

func TestEntites_WriteNewEntity(t *testing.T) {
	t.Run("write new entry", func(t *testing.T) {
		skipNoneDev(t)

		downstream, storage, client, err := getDownStreamApi()
		if err != nil {
			t.Fatal("not init", err)
		}

		txn := storage.Txn(true)
		entityRepo := repo.NewEntityRepo(txn)

		entity := &model.Entity{
			UUID:   utils.UUID(),
			Name:   randomStr(),
			UserId: randomStr(),
		}

		err = entityRepo.Put(entity)

		if err != nil {
			t.Fatal("Not write entity", err)
		}

		err = txn.Commit()
		if err != nil {
			t.Fatal("does not commit", err)
		}

		txnProc := storage.Txn(true)

		actions, err := downstream.ProcessObject(storage, txnProc, entity)
		if err != nil {
			t.Fatal("process obj return error", err)
		}

		if len(actions) != 1 {
			t.Fatal("must be return 1 action returns", len(actions))
		}

		err = actions[0].Execute()
		if err != nil {
			t.Fatal("action returns error", err)
		}

		id, err := vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).EntityApi().GetID(entity.Name)
		if err != nil {
			t.Fatal("getting entity id returns error", err)
		}

		if id == "" {
			t.Fatal("empty entity id")
		}
	})
}

func TestEntites_WriteNewEntityAlias(t *testing.T) {
	t.Run("write new entry", func(t *testing.T) {
		skipNoneDev(t)

		downstream, storage, client, err := getDownStreamApi()
		if err != nil {
			t.Fatal("not init", err)
		}

		txn := storage.Txn(true)
		entityRepo := repo.NewEntityRepo(txn)

		entity := &model.Entity{
			UUID:   utils.UUID(),
			Name:   randomStr(),
			UserId: randomStr(),
		}

		err = entityRepo.Put(entity)

		if err != nil {
			t.Fatal("Not write entity", err)
		}

		err = txn.Commit()
		if err != nil {
			t.Fatal("does not commit", err)
		}

		err = vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).EntityApi().Create(entity.Name)
		if err != nil {
			t.Fatal("does not create entity", err)
		}

		entityAlias := &model.EntityAlias{
			UUID:   utils.UUID(),
			Name:   randomStr(),
			UserId: entity.UserId,
		}

		txnProc := storage.Txn(true)

		actions, err := downstream.ProcessObject(storage, txnProc, entityAlias)
		if err != nil {
			t.Fatal("process obj return error", err)
		}

		if len(actions) != 1 {
			t.Fatal("must be return 1 action returns", len(actions))
		}

		err = actions[0].Execute()
		if err != nil {
			t.Fatal("action returns error", err)
		}

		accessor, err := downstream.mountAccessorGetter.MountAccessor()
		if err != nil {
			t.Fatal("not getting accessor", err)
		}

		id, err := vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).AliasApi().FindAliasIDByName(entityAlias.Name, accessor)
		if err != nil {
			t.Fatal("getting entity id returns error", err)
		}

		if id == "" {
			t.Fatal("empty entity id")
		}
	})
}

func TestEntites_DeleteEntity(t *testing.T) {
	t.Run("delete entity", func(t *testing.T) {
		skipNoneDev(t)

		downstream, storage, client, err := getDownStreamApi()
		if err != nil {
			t.Fatal("not init", err)
		}

		txn := storage.Txn(true)
		entityRepo := repo.NewEntityRepo(txn)

		entity := &model.Entity{
			UUID:   utils.UUID(),
			Name:   randomStr(),
			UserId: randomStr(),
		}

		err = entityRepo.Put(entity)

		if err != nil {
			t.Fatal("Not write entity", err)
		}

		err = txn.Commit()
		if err != nil {
			t.Fatal("does not commit", err)
		}

		err = vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).EntityApi().Create(entity.Name)
		if err != nil {
			t.Fatal("does not create entity", err)
		}

		entityId, err := vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).EntityApi().GetID(entity.Name)
		if err != nil {
			t.Fatal("does not get entity id", err)
		}

		entityAlias := &model.EntityAlias{
			UUID:   utils.UUID(),
			Name:   randomStr(),
			UserId: entity.UserId,
		}

		accessor, err := downstream.mountAccessorGetter.MountAccessor()
		if err != nil {
			t.Fatal("not getting accessor", err)
		}

		err = vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).AliasApi().Create(entityAlias.Name, entityId, accessor)
		if err != nil {
			t.Fatal("not create entity alias 1", err)
		}

		txnProc := storage.Txn(true)
		actions, err := downstream.ProcessObjectDelete(storage, txnProc, entity)

		if len(actions) != 1 {
			t.Fatal("must be return 1 action returns", len(actions))
		}

		err = actions[0].Execute()
		if err != nil {
			t.Fatal("action returns error", err)
		}

		id, err := vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).EntityApi().GetID(entity.Name)
		if err != nil && !strings.Contains(err.Error(), "empty response in op") {
			t.Fatal("getting entity id returns error", err)
		}

		if id != "" {
			t.Fatal("entity not deleted")
		}

		id, err = vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).AliasApi().FindAliasIDByName(entityAlias.Name, accessor)
		if err != nil && !strings.Contains(err.Error(), "nil response") {
			t.Fatal("getting entity id returns error", err)
		}

		if id != "" {
			t.Fatal("entity alias not deleted")
		}

		actions, err = downstream.ProcessObjectDelete(storage, txnProc, entity)

		if len(actions) != 1 {
			t.Fatal("must be return 1 action returns", len(actions))
		}

		err = actions[0].Execute()
		if err != nil && !strings.Contains(err.Error(), "nil response") {
			t.Fatal("not idempotent delete", err)
		}
	})
}

func TestEntites_DeleteEntityAlias(t *testing.T) {
	t.Run("delete entity alias", func(t *testing.T) {
		skipNoneDev(t)

		downstream, storage, client, err := getDownStreamApi()
		if err != nil {
			t.Fatal("not init", err)
		}

		txn := storage.Txn(true)
		entityRepo := repo.NewEntityRepo(txn)

		entity := &model.Entity{
			UUID:   utils.UUID(),
			Name:   randomStr(),
			UserId: randomStr(),
		}

		err = entityRepo.Put(entity)

		if err != nil {
			t.Fatal("Not write entity", err)
		}

		err = txn.Commit()
		if err != nil {
			t.Fatal("does not commit", err)
		}

		err = vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).EntityApi().Create(entity.Name)
		if err != nil {
			t.Fatal("does not create entity", err)
		}

		entityId, err := vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).EntityApi().GetID(entity.Name)
		if err != nil {
			t.Fatal("does not get entity id", err)
		}

		entityAlias1 := &model.EntityAlias{
			UUID:   utils.UUID(),
			Name:   randomStr(),
			UserId: entity.UserId,
		}

		entityAlias2 := &model.EntityAlias{
			UUID:   utils.UUID(),
			Name:   randomStr(),
			UserId: entity.UserId,
		}

		accessor, err := downstream.mountAccessorGetter.MountAccessor()
		if err != nil {
			t.Fatal("not getting accessor", err)
		}

		err = vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).AliasApi().Create(entityAlias1.Name, entityId, accessor)
		if err != nil {
			t.Fatal("not create entity alias 1", err)
		}
		err = vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).AliasApi().Create(entityAlias2.Name, entityId, accessor)
		if err != nil {
			t.Fatal("not create entity alias 1", err)
		}

		txnProc := storage.Txn(true)
		actions, err := downstream.ProcessObjectDelete(storage, txnProc, entityAlias1)

		if len(actions) != 1 {
			t.Fatal("must be return 1 action returns", len(actions))
		}

		err = actions[0].Execute()
		if err != nil {
			t.Fatal("action returns error", err)
		}

		id, err := vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).AliasApi().FindAliasIDByName(entityAlias1.Name, accessor)
		if err != nil && !strings.Contains(err.Error(), "nil response") {
			t.Fatal("getting entity id returns error", err)
		}

		if id != "" {
			t.Fatal("entity not deleted")
		}

		id, err = vault_identity.NewIdentityAPI(client, hclog.NewNullLogger()).AliasApi().FindAliasIDByName(entityAlias2.Name, accessor)
		if err != nil && !strings.Contains(err.Error(), "nil response") {
			t.Fatal("getting entity id returns error", err)
		}

		if id == "" {
			t.Fatal("entity deleted")
		}

		actions, err = downstream.ProcessObjectDelete(storage, txnProc, entityAlias1)

		if len(actions) != 1 {
			t.Fatal("must be return 1 action returns", len(actions))
		}

		err = actions[0].Execute()
		if err != nil && !strings.Contains(err.Error(), "nil response") {
			t.Fatal("not idempotent delete", err)
		}
	})
}
