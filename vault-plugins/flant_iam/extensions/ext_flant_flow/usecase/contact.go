package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ContactService struct {
	clientUUID  model.ClientUUID
	clientRepo  *repo.ClientRepository
	projectRepo *iam_repo.ProjectRepository
	repo        *repo.ContactRepository
	userService *iam_usecase.UserService
}

func Contacts(db *io.MemoryStoreTxn, clientUUID model.ClientUUID) *ContactService {
	return &ContactService{
		clientUUID:  clientUUID,
		clientRepo:  repo.NewClientRepository(db),
		projectRepo: iam_repo.NewProjectRepository(db),
		repo:        repo.NewContactRepository(db),
		userService: iam_usecase.Users(db, clientUUID, consts.OriginFlantFlow),
	}
}

func (s *ContactService) Create(fc *model.FullContact) error {
	contact := fc.GetContact()
	if err := s.validateCredentials(contact); err != nil {
		return err
	}
	fc.Version = repo.NewResourceVersion()
	if err := s.userService.Create(&fc.User); err != nil {
		return err
	}
	contact.Version = fc.Version
	return s.repo.Create(contact)
}

func (s *ContactService) Update(updated *model.FullContact) error {
	contact := updated.GetContact()
	if err := s.validateCredentials(contact); err != nil {
		return err
	}
	stored, err := s.repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	if stored.Version != updated.Version {
		return consts.ErrBadVersion
	}

	if err = s.userService.Update(&updated.User); err != nil {
		return err
	}
	contact.Version = updated.Version
	return s.repo.Update(contact)
}

func (s *ContactService) Delete(id model.ContactUUID) error {
	err := s.userService.Delete(id)
	if err != nil {
		return err
	}
	user, err := s.userService.GetByID(id)
	if err != nil {
		return err
	}
	archiveMark := user.ArchiveMark
	return s.repo.Delete(id, archiveMark)
}

func (s *ContactService) GetByID(id model.ContactUUID) (*model.FullContact, error) {
	user, err := s.userService.GetByID(id)
	if err != nil {
		return nil, err
	}
	contact, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return makeFullContact(user, contact)
}

func (s *ContactService) List(showArchived bool) ([]*model.FullContact, error) {
	cs, err := s.repo.List(s.clientUUID, showArchived)
	if err != nil {
		return nil, err
	}
	result := make([]*model.FullContact, len(cs))
	for i := range cs {
		user, err := s.userService.GetByID(cs[i].UserUUID)
		if err != nil {
			return nil, err
		}
		result[i], err = makeFullContact(user, cs[i])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *ContactService) Restore(id model.ContactUUID) (*model.FullContact, error) {
	user, err := s.userService.Restore(id)
	if err != nil {
		return nil, err
	}
	contact, err := s.repo.Restore(id)
	if err != nil {
		return nil, err
	}
	return makeFullContact(user, contact)
}

func (s *ContactService) validateCredentials(contact *model.Contact) error {
	for projectUUID, contactRole := range contact.Credentials {
		if err := s.validateProjectUUID(projectUUID); err != nil {
			return err
		}
		if _, ok := model.ContactRoles[contactRole]; !ok {
			return fmt.Errorf("%w: contact role not allowed: %s", consts.ErrInvalidArg, contactRole)
		}
	}
	return nil
}

func (s *ContactService) validateProjectUUID(uuid iam.ProjectUUID) error {
	_, err := s.projectRepo.GetByID(uuid)
	if errors.Is(err, consts.ErrNotFound) {
		return fmt.Errorf("%w: project with uuid:%s not found", consts.ErrInvalidArg, uuid)
	}
	return err
}

func makeFullContact(user *iam.User, contact *model.Contact) (*model.FullContact, error) {
	if user == nil || contact == nil {
		return nil, consts.ErrNilPointer
	}
	return &model.FullContact{
		User:        *user,
		Credentials: contact.Credentials,
	}, nil
}
