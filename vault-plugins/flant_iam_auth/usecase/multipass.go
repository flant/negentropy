package usecase

import (
	"fmt"

	"github.com/hashicorp/go-hclog"

	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	jwt_usecases "github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
)

type Multipass struct {
	MultipassRepo    *iam_repo.MultipassRepository
	GenMultipassRepo *repo.MultipassGenerationNumberRepository
	JwtController    *jwt.Controller

	Logger hclog.Logger
}

func (m *Multipass) GetWithGeneration(uuid string) (*iam_model.Multipass, *model.MultipassGenerationNumber, error) {
	multipass, err := m.MultipassRepo.GetByID(uuid)
	if err != nil {
		return nil, nil, err
	}

	if multipass == nil || multipass.Archived() {
		return nil, nil, fmt.Errorf("not found multipass")
	}

	m.Logger.Debug(fmt.Sprintf("Try to get multipass generation number %s", uuid))
	multipassGen, err := m.GenMultipassRepo.GetByID(multipass.UUID)
	if err != nil {
		return nil, nil, err
	}

	if multipassGen == nil {
		m.Logger.Error(fmt.Sprintf("Not found multipass generation number %s", uuid))
		return nil, nil, fmt.Errorf("not found multipass gen")
	}

	return multipass, multipassGen, nil
}

func (m *Multipass) IssueNewMultipassGeneration(txn *io.MemoryStoreTxn, multipassUUID iam_model.MultipassUUID,
	ownerType iam_model.MultipassOwnerType, ownerUUID iam_model.OwnerUUID) (string, error) {
	mp, gen, err := m.GetWithGeneration(multipassUUID)
	if err != nil {
		return "", err
	}
	if mp.OwnerType != ownerType || mp.OwnerUUID != ownerUUID {
		return "", consts.ErrNotFound // passed multipass (uuid) belongs to another user
	}

	nextGen := gen.GenerationNumber + 1

	tokenStr, err := m.JwtController.IssueMultipass(txn, &jwt_usecases.PrimaryTokenOptions{
		TTL:  mp.TTL,
		UUID: mp.UUID,
		JTI: jwt_usecases.TokenJTI{
			Generation: nextGen,
			SecretSalt: mp.Salt,
		},
	})
	if err != nil {
		return "", err
	}

	gen.GenerationNumber = nextGen
	err = m.GenMultipassRepo.Update(gen)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}
