package iam

import (
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RestoreObjectHandler struct{}

func NewRestoreObjectHandler(_ *io.MemoryStoreTxn) *RestoreObjectHandler {
	return &RestoreObjectHandler{}
}

func (r *RestoreObjectHandler) HandleUser(_ *iam.User) error {
	return nil
}

func (r *RestoreObjectHandler) HandleServiceAccount(_ *iam.ServiceAccount) error {
	return nil
}
