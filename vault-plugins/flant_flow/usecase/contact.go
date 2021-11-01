package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_flow/iam_client"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ContactService struct {
	clientUUID model.ClientUUID

	clientRepo   *iam_repo.ClientRepository
	contactsRepo *iam_repo.ContactRepository
	userClient   iam_client.Users
}

func Contacts(db *io.MemoryStoreTxn, clientUUID model.ClientUUID, userClient iam_client.Users) *ContactService {
	return &ContactService{
		clientUUID: clientUUID,

		clientRepo:   iam_repo.NewClientRepository(db),
		contactsRepo: iam_repo.NewContactRepository(db),
		userClient:   userClient,
	}
}

func (s *ContactService) Create(contact *model.Contact) error {
	client, err := s.clientRepo.GetByID(s.clientUUID)
	if err != nil {
		return err
	}

	contact.Version = iam_repo.NewResourceVersion()
	contact.FullIdentifier = contact.Identifier + "@" + client.Identifier
	if contact.Origin == "" {
		return model.ErrBadOrigin
	}
	return s.contactsRepo.Create(contact)
}

func (s *ContactService) GetByID(id model.ContactUUID) (*model.Contact, error) {
	return s.contactsRepo.GetByID(id)
}

func (s *ContactService) List(showArchived bool) ([]*model.Contact, error) {
	return s.contactsRepo.List(s.clientUUID, showArchived)
}

func (s *ContactService) Update(contact *model.Contact) error {
	stored, err := s.contactsRepo.GetByID(contact.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != s.clientUUID {
		return model.ErrNotFound
	}
	if stored.Version != contact.Version {
		return model.ErrBadVersion
	}
	if stored.Origin != contact.Origin {
		return model.ErrBadOrigin
	}

	client, err := s.clientRepo.GetByID(s.clientUUID)
	if err != nil {
		return err
	}

	// Update
	contact.TenantUUID = s.clientUUID
	contact.Version = iam_repo.NewResourceVersion()
	contact.FullIdentifier = contact.Identifier + "@" + client.Identifier

	// Preserve fields, that are not always accessible from the outside, e.g. from HTTP API
	if contact.Extensions == nil {
		contact.Extensions = stored.Extensions
	}
	return s.contactsRepo.Update(contact)
}

func (s *ContactService) Delete(id model.ContactUUID) error {
	_, err := s.contactsRepo.GetByID(id)
	if err != nil {
		return err
	}
	archivingTimestamp, archivingHash := ArchivingLabel()

	return s.contactsRepo.Delete(id, archivingTimestamp, archivingHash)
}

func (s *ContactService) Restore(id model.ContactUUID) (*model.Contact, error) {
	return s.contactsRepo.Restore(id)
}
