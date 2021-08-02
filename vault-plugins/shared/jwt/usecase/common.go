package usecase

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"

	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

type Depends struct {
	Now      func() time.Time
	idGetter func() (string, error)
	Logger   hclog.Logger
}

func NewDeps(idGetter func() (string, error), logger hclog.Logger, now func() time.Time) *Depends {
	return &Depends{
		Now:      now,
		idGetter: idGetter,
		Logger:   logger,
	}
}

func (b *Depends) ConfigRepo(tnx *sharedio.MemoryStoreTxn) *model.ConfigRepo {
	return model.NewConfigRepo(tnx)
}

func (b *Depends) JwksRepo(db *sharedio.MemoryStoreTxn) (*model.JWKSRepo, error) {
	id, err := b.idGetter()
	if err != nil {
		b.Logger.Error(fmt.Sprintf("Can not get plugin id: %v", err), "err", err)
		return nil, err
	}

	jwksId := utils.ShaEncode(id)
	return model.NewJWKSRepo(db, jwksId), nil
}

func (b *Depends) StateRepo(db *sharedio.MemoryStoreTxn) (*model.StateRepo, error) {
	return model.NewStateRepo(db), nil
}

func (b *Depends) KeyPairsService(db *sharedio.MemoryStoreTxn) (*KeyPairService, error) {
	jwksRepo, err := b.JwksRepo(db)
	if err != nil {
		return nil, err
	}

	c, err := b.ConfigRepo(db).Get()
	if err != nil {
		return nil, err
	}

	stateRepo, err := b.StateRepo(db)
	if err != nil {
		return nil, err
	}

	return NewKeyPairService(stateRepo, jwksRepo, c, b.Now, b.Logger), nil
}
