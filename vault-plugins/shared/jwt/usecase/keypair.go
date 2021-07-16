package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ed25519"
	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

type KeyPairService struct {
	stateRepo *model.StateRepo
	config    *model.Config
	now       func() time.Time
}

func NewKeyPairService(stateRepo *model.StateRepo, config *model.Config, now func() time.Time) *KeyPairService {
	return &KeyPairService{
		config: config,
		stateRepo: stateRepo,
		now: now,
	}
}

func (s *KeyPairService) EnableJwt() error {
	enabled, err := s.stateRepo.IsEnabled()
	if enabled {
		return nil
	}

	kp, err := s.stateRepo.GetKeyPair()
	if err != nil {
		return err
	}

	if kp == nil {
		err := s.GenerateOrRotateKeys()
		if err != nil {
			return err
		}
	}

	return s.stateRepo.SetEnabled(true)
}

func (s *KeyPairService) DisableJwt() error {
	enabled, err := s.stateRepo.IsEnabled()
	if !enabled {
		return nil
	}

	err = s.stateRepo.SetKeyPair(nil)
	if err != nil {
		return err
	}

	// todo Do need anounce delete key pairs?

	return s.stateRepo.SetEnabled(false)
}

func (s *KeyPairService) ForceRotateKeys() error {
	priv, pub, err := generateKeys(s.config)
	if err != nil {
		return err
	}

	err = s.stateRepo.SetKeyPair(&model.KeyPair{
		PublicKeys:  &model.JSONWebKeySet{Keys: []*model.JSONWebKey{pub}},
		PrivateKeys: &model.JSONWebKeySet{Keys: []*model.JSONWebKey{priv}},
	})

	err = s.stateRepo.SetLastRotationTime(s.now())
	if err != nil {
		return err
	}

	return nil
}

func (s *KeyPairService) RunPeriodicalRotateKeys() error {
	shouldRotate, shouldPublish, err := s.shouldRotateOrPublish()
	if err != nil {
		return err
	}

	if shouldRotate {
		err := s.removeFirstKey()
		if err != nil {
			return err
		}
	} else if shouldPublish {
		err := s.GenerateOrRotateKeys()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *KeyPairService) modifyKeys(modify func(*model.JSONWebKeySet, *model.JSONWebKeySet) error) error {
	keyPair, err := s.stateRepo.GetKeyPair()
	if err != nil {
		return err
	}

	publicKeySet := model.JSONWebKeySet{}
	privateSet := model.JSONWebKeySet{}

	if keyPair != nil {
		publicKeySet = *keyPair.PublicKeys
		privateSet = *keyPair.PrivateKeys
	} else {
		keyPair = &model.KeyPair{
			PublicKeys: &model.JSONWebKeySet{},
			PrivateKeys: &model.JSONWebKeySet{},
		}
	}

	err = modify(&privateSet, &publicKeySet)
	if err != nil {
		return err
	}

	keyPair.PrivateKeys = &privateSet
	keyPair.PublicKeys = &publicKeySet

	return s.stateRepo.SetKeyPair(keyPair)
}

// GenerateOrRotateKeys generates a new keypair and adds it to keys in the storage
func (s *KeyPairService) GenerateOrRotateKeys() error {
	err := s.modifyKeys(func(privateSet, pubicKeySet *model.JSONWebKeySet) error {
		priv, pub, err := generateKeys(s.config)
		if err != nil {
			return err
		}

		privateSet.Keys = append(privateSet.Keys, priv)
		if len(privateSet.Keys) > 2 {
			privateSet.Keys = privateSet.Keys[1:len(privateSet.Keys)]
		}
		pubicKeySet.Keys = append(pubicKeySet.Keys, pub)
		if len(pubicKeySet.Keys) > 2 {
			pubicKeySet.Keys = pubicKeySet.Keys[1:len(pubicKeySet.Keys)]
		}

		return nil
	})

	if err != nil {
		return err
	}

	return s.stateRepo.SetLastRotationTime(s.now())
}

// removeFirstKey remove the key if there are more than one
func (s *KeyPairService) removeFirstKey() error {
	return s.modifyKeys(func(privateSet, pubicKeySet *model.JSONWebKeySet) error {
		if len(privateSet.Keys) == 2 {
			privateSet.Keys = privateSet.Keys[1:]
		}
		if len(pubicKeySet.Keys) == 2 {
			pubicKeySet.Keys = pubicKeySet.Keys[1:]
		}
		return nil
	})
}

func (s *KeyPairService) shouldRotateOrPublish() (bool, bool, error) {
	lastRotation, err := s.stateRepo.GetLastRotationTime()
	if err != nil {
		return false, false, err
	}

	now := s.now()
	rotateEvery := s.config.RotationPeriod
	publishKeyBefore := s.config.PreliminaryAnnouncePeriod

	if !lastRotation.Add(rotateEvery).After(now) {
		return true, false, nil
	} else if !lastRotation.Add(rotateEvery).Add(-publishKeyBefore).After(now) {
		return false, true, nil
	}
	return false, false, nil
}

// newUUID generates random string
func newUUID() string {
	u := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, u); err != nil {
		panic(err)
	}

	u[8] = (u[8] | 0x80) & 0xBF
	u[6] = (u[6] | 0x40) & 0x4F

	return hex.EncodeToString(u)
}

func generateKeys(conf *model.Config) (*model.JSONWebKey, *model.JSONWebKey, error) {
	pubKey, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("gen ecdsa key: %v", err)
	}

	genTime := time.Now()
	// TODO specify endTime. Now it not used but it has in specification
	endLifeTime := time.Now().Add(2 * conf.RotationPeriod)

	priv := model.JSONWebKey{
		JSONWebKey: jose.JSONWebKey{
			Key:       key,
			KeyID:     newUUID(),
			Algorithm: string(jose.EdDSA),
			Use:       "sig",
		},

		GenerateTime: genTime,
		EndLifeTime:  endLifeTime,
	}

	pub := model.JSONWebKey{
		JSONWebKey: jose.JSONWebKey{
			Key:       pubKey,
			KeyID:     newUUID(),
			Algorithm: string(jose.EdDSA),
			Use:       "sig",
		},

		GenerateTime: genTime,
		EndLifeTime:  endLifeTime,
	}

	return &priv, &pub, nil
}
