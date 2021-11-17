package usecase

import (
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func ArchivingLabel() (model.UnixTime, int64) {
	archivingTime := time.Now().Unix()
	archivingHash := rand.Int63n(archivingTime)
	return archivingTime, archivingHash
}
