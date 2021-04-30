package iam

import (
	"fmt"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

type ObjectHandler struct {
	entityRepo     *repo.EntityRepo
	eaRepo         *repo.EntityAliasRepo
	authSourceRepo *repo.AuthSourceRepo
}

func NewObjectHandler(txn *io.MemoryStoreTxn) *ObjectHandler {
	return &ObjectHandler{
		entityRepo:     repo.NewEntityRepo(txn),
		eaRepo:         repo.NewEntityAliasRepo(txn),
		authSourceRepo: repo.NewAuthSourceRepo(txn),
	}
}

func (h *ObjectHandler) HandleUser(user *iam.User) error {
	err := h.putNewEntity(user.FullIdentifier, user.UUID)

	err = h.authSourceRepo.Iter(func(source *model.AuthSource) (bool, error) {
		var name string
		switch source.EntityAliasName {
		case model.EntityAliasNameEmail:
			name = user.Email
		case model.EntityAliasNameFullIdentifier:
			name = user.FullIdentifier
		case model.EntityAliasNameUUID:
			name = user.UUID
		default:
			return false, fmt.Errorf("incorrect source entity alias name %s", source.EntityAliasName)
		}

		err = h.putNewEntityAlias(h.authSourceId(source), name, user.Identifier)
		if err != nil {
			return false, err
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (h *ObjectHandler) HandleServiceAccount(sa *iam.ServiceAccount) error {
	err := h.putNewEntity(sa.FullIdentifier, sa.UUID)

	err = h.authSourceRepo.Iter(func(source *model.AuthSource) (bool, error) {
		if !source.AllowServiceAccounts || source.EntityAliasName == model.EntityAliasNameEmail {
			return true, nil
		}

		var name string
		switch source.EntityAliasName {
		case model.EntityAliasNameFullIdentifier:
			name = sa.FullIdentifier
		case model.EntityAliasNameUUID:
			name = sa.UUID
		default:
			return false, fmt.Errorf("incorrect source entity alias name %s", source.EntityAliasName)
		}

		err = h.putNewEntityAlias(h.authSourceId(source), name, sa.Identifier)

		if err != nil {
			return false, err
		}

		return true, nil
	})

	return nil
}

func (h *ObjectHandler) putNewEntity(name string, userId string) error {
	entity, err := h.entityRepo.GetByUserId(userId)
	if err != nil {
		return err
	}

	if entity == nil {
		entity = &model.Entity{
			UUID:   utils.UUID(),
			UserId: userId,
		}
	}

	entity.Name = name

	return h.entityRepo.Put(entity)
}

func (h *ObjectHandler) putNewEntityAlias(sourceName string, name string, userId string) error {
	entityAlias, err := h.eaRepo.GetByUserID(userId, sourceName)
	if err != nil {
		return err
	}

	if entityAlias == nil {
		entityAlias = &model.EntityAlias{
			UUID:   utils.UUID(),
			UserId: userId,
		}
	}

	entityAlias.Name = name
	entityAlias.SourceIdentifier = sourceName

	return h.eaRepo.Put(entityAlias)
}

func (h *ObjectHandler) authSourceId(source *model.AuthSource) string {
	return source.Name
}
