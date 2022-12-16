package root

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/go-hclog"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ObjectHandler struct {
	logger hclog.Logger
}

func NewObjectHandler(parentLogger hclog.Logger) *ObjectHandler {
	return &ObjectHandler{
		logger: parentLogger.Named("RootSourceHandler"),
	}
}

func (h *ObjectHandler) HandleUser(txn io.Txn, user *iam_model.User) error {
	l := h.logger
	l.Debug("Handle new user. Create entity object", "identifier", user.FullIdentifier)
	entityRepo := repo.NewEntityRepo(txn)
	authSourceRepo := repo.NewAuthSourceRepo(txn)
	eaRepo := repo.NewEntityAliasRepo(txn)

	err := entityRepo.CreateForUser(user)
	if err != nil {
		return err
	}
	l.Debug("Entity object created for user", "identifier", user.FullIdentifier)

	err = authSourceRepo.Iter(true, func(source *model.AuthSource) (bool, error) {
		if source.OnlyServiceAccounts {
			l.Debug("skipped creating entity alias for user and source, due to source only for service_accounts",
				"identifier", user.FullIdentifier, "source", source.Name)
			return true, nil
		}
		l.Debug("Creating entity alias for user and source", "identifier", user.FullIdentifier, "source", source.Name)
		ea, err := eaRepo.CreateForUser(user, source)
		if errors.Is(err, repo.ErrEmptyEntityAliasName) {
			l.Debug("skipped creating entity alias for user and source, due to empty alias name error",
				"identifier", user.FullIdentifier, "source", source.Name, "error", err.Error())
			return true, nil
		}

		if err != nil {
			return false, err
		}

		l.Debug("Entity alias for user and source created", "identifier", user.FullIdentifier, "source", source.Name, "entityAliasUUID", ea.UUID)
		return true, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (h *ObjectHandler) HandleServiceAccount(txn io.Txn, sa *iam_model.ServiceAccount) error {
	l := h.logger
	entityRepo := repo.NewEntityRepo(txn)
	eaRepo := repo.NewEntityAliasRepo(txn)
	authSourceRepo := repo.NewAuthSourceRepo(txn)

	err := entityRepo.CreateForSA(sa)
	if err != nil {
		return err
	}

	err = authSourceRepo.Iter(true, func(source *model.AuthSource) (bool, error) {
		ea, err := eaRepo.CreateForSA(sa, source)
		if err != nil {
			return false, err
		}
		l.Debug("Entity alias for sa and source created", "identifier", sa.FullIdentifier, "source", source.Name, "entityAliasUUID", ea.UUID)
		return true, nil
	})

	return nil
}

func (h *ObjectHandler) HandleDeleteUser(txn io.Txn, uuid string) error {
	return h.deleteEntityWithAliases(txn, uuid)
}

func (h *ObjectHandler) HandleDeleteServiceAccount(txn io.Txn, uuid string) error {
	return h.deleteEntityWithAliases(txn, uuid)
}

func (h *ObjectHandler) HandleMultipass(txn io.Txn, mp *iam_model.Multipass) error {
	l := h.logger
	l.Debug(fmt.Sprintf("Handle multipass %s", mp.UUID))
	multipassGenRepo := repo.NewMultipassGenerationNumberRepository(txn)
	genNum, err := multipassGenRepo.GetByID(mp.UUID)
	if err != nil {
		return err
	}

	if genNum != nil {
		l.Debug(fmt.Sprintf("Found multipass generation number %s. Skip create", mp.UUID))
		return nil
	}

	genNum = &model.MultipassGenerationNumber{
		UUID:             mp.UUID,
		GenerationNumber: 0,
	}

	l.Debug(fmt.Sprintf("Try to create generation number for multipass %s", mp.UUID))
	return multipassGenRepo.Create(genNum)
}

func (h *ObjectHandler) HandleDeleteMultipass(txn io.Txn, uuid string) error {
	h.logger.Debug(fmt.Sprintf("Handle delete multipass multipass %s. Try to delete multipass gen number", uuid))
	multipassGenRepo := repo.NewMultipassGenerationNumberRepository(txn)
	return multipassGenRepo.Delete(uuid)
}

func (h *ObjectHandler) deleteEntityWithAliases(txn io.Txn, uuid string) error {
	// begin delete entity aliases
	entityRepo := repo.NewEntityRepo(txn)
	eaRepo := repo.NewEntityAliasRepo(txn)

	err := eaRepo.GetAllForUser(uuid, func(alias *model.EntityAlias) (bool, error) {
		err := eaRepo.DeleteByID(alias.UUID)
		if err != nil {
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	return entityRepo.DeleteForUser(uuid)
}

type ModelHandler interface {
	HandleUser(txn io.Txn, user *iam_model.User) error
	HandleDeleteUser(txn io.Txn, uuid string) error

	HandleMultipass(txn io.Txn, mp *iam_model.Multipass) error
	HandleDeleteMultipass(txn io.Txn, uuid string) error

	HandleServiceAccount(txn io.Txn, sa *iam_model.ServiceAccount) error
	HandleDeleteServiceAccount(txn io.Txn, uuid string) error
}

func HandleNewMessageIamRootSource(txn io.Txn, handler ModelHandler, msg io.MsgDecoded) error {
	isDelete := msg.IsDeleted()

	var inputObject interface{}
	var entityHandler func() error

	objID := msg.ID

	switch msg.Type {
	case iam_model.UserType:
		user := &iam_model.User{}
		user.UUID = objID
		inputObject = user
		// dont call here because we need unmarshal and add object in mem storage before handler
		if isDelete {
			entityHandler = func() error {
				return handler.HandleDeleteUser(txn, objID)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleUser(txn, user)
			}
		}

	case iam_model.ServiceAccountType:
		sa := &iam_model.ServiceAccount{}
		sa.UUID = objID
		inputObject = sa
		if isDelete {
			entityHandler = func() error {
				return handler.HandleDeleteServiceAccount(txn, objID)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleServiceAccount(txn, sa)
			}
		}
	case iam_model.ProjectType:
		p := &iam_model.Project{}
		p.UUID = objID
		inputObject = p

	case iam_model.TenantType:
		t := &iam_model.Tenant{}
		t.UUID = objID
		inputObject = t

	case iam_model.FeatureFlagType:
		t := &iam_model.FeatureFlag{}
		t.Name = objID
		inputObject = t

	case iam_model.GroupType:
		t := &iam_model.Group{}
		t.UUID = objID
		inputObject = t

	case iam_model.RoleType:
		t := &iam_model.Role{}
		t.Name = objID
		inputObject = t

	case iam_model.RoleBindingType:
		t := &iam_model.RoleBinding{}
		t.UUID = objID
		inputObject = t

	case iam_model.RoleBindingApprovalType:
		t := &iam_model.RoleBindingApproval{}
		t.UUID = objID
		inputObject = t

	case iam_model.MultipassType:
		mp := &iam_model.Multipass{}
		mp.UUID = objID
		inputObject = mp
		if isDelete {
			entityHandler = func() error {
				return handler.HandleDeleteMultipass(txn, objID)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleMultipass(txn, mp)
			}
		}

	case iam_model.ServiceAccountPasswordType:
		t := &iam_model.ServiceAccountPassword{}
		t.UUID = objID
		inputObject = t

	case iam_model.IdentitySharingType:
		t := &iam_model.IdentitySharing{}
		t.UUID = objID
		inputObject = t

	case ext_model.ServerType:
		inputObject = &ext_model.Server{}

	default:
		return nil
	}

	table := msg.Type

	if isDelete {
		err := txn.Delete(table, inputObject)
		if err != nil {
			return err
		}

		if entityHandler != nil {
			return entityHandler()
		}

		return nil
	}

	err := json.Unmarshal(msg.Data, inputObject)
	if err != nil {
		return err
	}

	err = txn.Insert(table, inputObject)
	if err != nil {
		return err
	}

	if entityHandler != nil {
		return entityHandler()
	}

	return nil
}
