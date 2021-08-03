package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/crypto/ed25519"
	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

type KeyPairService struct {
	stateRepo *model.StateRepo
	jwksRepo  *model.JWKSRepo
	config    *model.Config
	now       func() time.Time
	logger    hclog.Logger
}

func NewKeyPairService(stateRepo *model.StateRepo, jwks *model.JWKSRepo, config *model.Config, now func() time.Time, logger hclog.Logger) *KeyPairService {
	return &KeyPairService{
		config:    config,
		stateRepo: stateRepo,
		now:       now,
		jwksRepo:  jwks,
		logger:    logger,
	}
}

func (s *KeyPairService) EnableJwt() error {
	enabled, err := s.stateRepo.IsEnabled()
	if err != nil {
		return err
	}

	if enabled {
		return nil
	}

	err = s.ForceRotateKeys()
	if err != nil {
		return err
	}

	return s.stateRepo.SetEnabled(true)
}

func (s *KeyPairService) DisableJwt() error {
	enabled, err := s.stateRepo.IsEnabled()
	if !enabled {
		return nil
	}

	err = s.modifyKeys(func(pair **model.KeyPair) (bool, error) {
		*pair = nil
		return true, err
	})

	if err != nil {
		return err
	}

	return s.stateRepo.SetEnabled(false)
}

func (s *KeyPairService) ForceRotateKeys() error {
	err := s.generateNewKey()
	if err != nil {
		return err
	}

	return s.rotateKeys(true)
}

func (s *KeyPairService) RunPeriodicalRotateKeys() error {
	shouldRotate, shouldGenerate, err := s.shouldRotateOrGenNew()
	if err != nil {
		return err
	}

	s.logger.Debug(fmt.Sprintf("shouldRotate=%v shouldGenerate=%v", shouldRotate, shouldGenerate))

	if shouldRotate {
		err = s.rotateKeys(false)
	} else if shouldGenerate {
		err = s.generateNewKey()
	}

	return err
}

func (s *KeyPairService) modifyKeys(modify func(pair **model.KeyPair) (bool, error)) error {
	keyPair, err := s.stateRepo.GetKeyPair()
	if err != nil {
		return err
	}

	if keyPair == nil {
		keyPair = &model.KeyPair{
			PublicKeys:  &model.JSONWebKeySet{},
			PrivateKeys: &model.JSONWebKeySet{},
		}
	}

	modified, err := modify(&keyPair)
	if err != nil {
		return err
	}

	if !modified {
		return nil
	}

	err = s.stateRepo.SetKeyPair(keyPair)
	if err != nil {
		return err
	}

	if keyPair == nil {
		return s.jwksRepo.DeleteOwn()
	}

	return s.jwksRepo.UpdateOwn(keyPair.PublicKeys)
}

// generateNewKey generates a new keypair and adds it to keys in the storage
func (s *KeyPairService) generateNewKey() error {
	return s.modifyKeys(func(keyPair **model.KeyPair) (bool, error) {
		privateSet := (*keyPair).PrivateKeys
		pubicKeySet := (*keyPair).PublicKeys

		l := len(privateSet.Keys)
		if l == 2 {
			return false, nil
		} else if l > 2 {
			return false, fmt.Errorf("incorrect private keys count %v", l)
		}

		priv, pub, err := generateKeys(s.config)
		if err != nil {
			return false, err
		}

		privateSet.Keys = append(privateSet.Keys, priv)
		pubicKeySet.Keys = append(pubicKeySet.Keys, pub)

		return true, nil
	})
}

// rotateKeys remove the key if there are more than one
func (s *KeyPairService) rotateKeys(force bool) error {
	return s.modifyKeys(func(keyPair **model.KeyPair) (bool, error) {
		privateSet := (*keyPair).PrivateKeys
		pubicKeySet := (*keyPair).PublicKeys

		modify := false
		if len(privateSet.Keys) == 2 {
			privateSet.Keys = privateSet.Keys[1:]
			modify = true
		}

		// see readme, why 3
		if len(pubicKeySet.Keys) == 3 {
			pubicKeySet.Keys = pubicKeySet.Keys[1:]
			modify = true

		}

		if force || modify {
			err := s.stateRepo.SetLastRotationTime(s.now())
			if err == nil {
				return false, err
			}
		}

		return modify, nil
	})
}

func (s *KeyPairService) shouldRotateOrGenNew() (rotate, generate bool, err error) {
	lastRotation, err := s.stateRepo.GetLastRotationTime()
	if err != nil {
		return false, false, err
	}

	now := s.now()
	rotateEvery := s.config.RotationPeriod
	publishKeyBefore := s.config.PreliminaryAnnouncePeriod

	timeForRotate := lastRotation.Add(rotateEvery)
	timeForGenerate := lastRotation.Add(rotateEvery).Add(-publishKeyBefore)

	if !timeForRotate.After(now) {
		return true, false, nil
	} else if !timeForGenerate.After(now) {
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

	// TODO specify start/endTime. Now it not used but it has in specification
	startTime := time.Unix(genTime.Unix(), 0).Add(conf.PreliminaryAnnouncePeriod)
	endLifeTime := time.Unix(startTime.Unix(), 0).Add(2 * conf.RotationPeriod)

	priv := model.JSONWebKey{
		JSONWebKey: jose.JSONWebKey{
			Key:       key,
			KeyID:     newUUID(),
			Algorithm: string(jose.EdDSA),
			Use:       "sig",
		},

		GenerateTime: genTime,
		StartTime:    startTime,
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
		StartTime:    startTime,
		EndLifeTime:  endLifeTime,
	}

	return &priv, &pub, nil
}
