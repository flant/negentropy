package usecase

import (
	"fmt"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
	"github.com/hashicorp/go-hclog"
	"time"
)

type Depends struct {
	Now      func() time.Time
	idGetter func() (string, error)
	logger   hclog.Logger
}

func NewDeps(idGetter func() (string, error), logger hclog.Logger, now func() time.Time) *Depends {
	return &Depends{
		Now:      now,
		idGetter: idGetter,
		logger:   logger,
	}
}

func (b *Depends) ConfigRepo(tnx *sharedio.MemoryStoreTxn) *model.ConfigRepo {
	return model.NewConfigRepo(tnx)
}

func (b *Depends) JwksRepo(db *sharedio.MemoryStoreTxn) (*model.JWKSRepo, error) {
	id, err := b.idGetter()
	if err != nil {
		b.logger.Error(fmt.Sprintf("Can not get plugin id: %v", err), "err", err)
		return nil, err
	}

	jwksId := utils.ShaEncode(id)
	return model.NewJWKSRepo(db, jwksId), nil
}

func (b *Depends) StateRepo(db *sharedio.MemoryStoreTxn) (*model.StateRepo, error) {
	jwksRepo, err := b.JwksRepo(db)
	if err != nil {
		return nil, err
	}
	return model.NewStateRepo(db, jwksRepo), nil
}

func (b *Depends) KeyPairsService(db *sharedio.MemoryStoreTxn) (*KeyPairService, error) {
	c, err := b.ConfigRepo(db).Get()
	if err != nil {
		return nil, err
	}

	stateRepo, err := b.StateRepo(db)
	if err != nil {
		return nil, err
	}

	return NewKeyPairService(stateRepo, c, b.Now), nil
}
