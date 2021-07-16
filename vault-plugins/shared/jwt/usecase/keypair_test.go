package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/test"
)

func runJWKSTest(t *testing.T, b logical.Backend, storage logical.Storage) []jose.JSONWebKey {
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "jwks",
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	test.RequireValidResponse(t, resp, err)

	keys := resp.Data["keys"].([]jose.JSONWebKey)
	require.NoError(t, err, "error on keys unmarshall")

	return keys
}

func getTestService(t *testing.T) (*KeyPairService, *io.MemoryStoreTxn) {
	config := &logical.BackendConfig{
		StorageView: &logical.InmemStorage{},
	}
	storage := test.GetStorage(t, config)

	tx := storage.Txn(true)
	state := model.NewStateRepo(tx)
	jwks := model.NewJWKSRepo(tx, "id")
	cnf := model.DefaultConfig()

	return NewKeyPairService(state, jwks, cnf, time.Now, hclog.NewNullLogger()), tx
}

func Test_CheckRotation(t *testing.T) {
	s, tx := getTestService(t)
	defer tx.Abort()

	s.config.RotationPeriod = 10 * time.Second
	s.config.PreliminaryAnnouncePeriod = 2 * time.Second

	err := s.stateRepo.SetLastRotationTime(time.Unix(1, 0))
	require.NoError(t, err)

	t.Run("should not rotate and should not generate new", func(t *testing.T) {
		for _, st := range []int64{2, 8} {
			s.now = test.GetNowFn(st)
			shouldRotate, shouldGenNew, err := s.shouldRotateOrGenNew()

			require.NoError(t, err)
			require.False(t, shouldRotate)
			require.False(t, shouldGenNew)
		}
	})

	t.Run("should generate new but should not rotate", func(t *testing.T) {
		for _, st := range []int64{9, 10} {
			s.now = test.GetNowFn(st)
			shouldRotate, shouldGenNew, err := s.shouldRotateOrGenNew()

			require.NoError(t, err)
			require.True(t, shouldGenNew)
			require.False(t, shouldRotate)
		}
	})

	t.Run("should rotate but should not generate new", func(t *testing.T) {
		s.now = test.GetNowFn(11)
		shouldRotate, shouldGenNew, err := s.shouldRotateOrGenNew()

		require.NoError(t, err)
		require.False(t, shouldGenNew)
		require.True(t, shouldRotate)
	})
}

func TestJWKSRotation(t *testing.T) {
	s, tx := getTestService(t)
	defer tx.Abort()

	creteIntervals := func(rotateTime int64) (afterRotation, genNew, afterGenNew, nextRotation, afterNextRotation int64) {
		rotPeriod := int64(s.config.RotationPeriod.Seconds())
		annPeriod := int64(s.config.PreliminaryAnnouncePeriod.Seconds())

		afterRotation = rotateTime + 1
		genNew = rotateTime + (rotPeriod - annPeriod) + 1
		afterGenNew = genNew + 1
		nextRotation = rotateTime + rotPeriod + 1
		afterNextRotation = nextRotation + 1

		return afterRotation, genNew, afterGenNew, nextRotation, afterNextRotation
	}

	const initTimeRotation = 1

	s.config.RotationPeriod = 12 * time.Second
	s.config.PreliminaryAnnouncePeriod = 3 * time.Second
	s.now = test.GetNowFn(initTimeRotation)

	err := s.EnableJwt()
	require.NoError(t, err)

	getJWKS := func() []jose.JSONWebKey {
		k, err := s.jwksRepo.GetSet()
		require.NoError(t, err)

		return k
	}

	runPeriodicalRotate := func(time int64) {
		s.now = test.GetNowFn(time)
		err = s.RunPeriodicalRotateKeys()
		require.NoError(t, err)
	}

	getPrivateKeys := func() []*model.JSONWebKey {
		pair, err := s.stateRepo.GetKeyPair()
		require.NoError(t, err)
		return pair.PrivateKeys.Keys
	}

	assertLenPublicAndPrivateKeys := func(pubLen, privLen int) ([]jose.JSONWebKey, []*model.JSONWebKey) {
		currentKeys := getJWKS()
		require.Len(t, currentKeys, pubLen)

		privKeys := getPrivateKeys()
		require.Len(t, privKeys, privLen)

		return currentKeys, privKeys
	}

	assertEqualsKeysAfterRotate := func(pub []jose.JSONWebKey, priv []*model.JSONWebKey) {
		keys := getJWKS()
		diff := deep.Equal(pub, keys)
		require.Nil(t, diff)

		privKeysNow := getPrivateKeys()
		diff = deep.Equal(priv, privKeysNow)
		require.Nil(t, diff)
	}

	curRotationTime := int64(0)

	t.Run("first rotation cycle after enable", func(t *testing.T) {
		afterRotation, genNew, afterGenNew, nextRotation, afterNextRotation := creteIntervals(initTimeRotation)
		curRotationTime = nextRotation

		t.Run("should not rotate keys after already now enable jwt", func(t *testing.T) {
			pubKeys, privKeys := assertLenPublicAndPrivateKeys(1, 1)

			runPeriodicalRotate(afterRotation)

			assertEqualsKeysAfterRotate(pubKeys, privKeys)
		})

		t.Run("should generate keys on gen time", func(t *testing.T) {
			pub, priv := assertLenPublicAndPrivateKeys(1, 1)

			runPeriodicalRotate(genNew)

			nowPub, nowPriv := assertLenPublicAndPrivateKeys(2, 2)

			diff := deep.Equal(pub[0], nowPub[0])
			require.Nil(t, diff)

			diff = deep.Equal(priv[0], nowPriv[0])
			require.Nil(t, diff)
		})

		t.Run("should not rotate keys after generation", func(t *testing.T) {
			pubKeys, privKeys := assertLenPublicAndPrivateKeys(2, 2)

			runPeriodicalRotate(afterGenNew)

			assertEqualsKeysAfterRotate(pubKeys, privKeys)
		})

		t.Run("should rotate private keys but stay public keys", func(t *testing.T) {
			pubKeys, _ := assertLenPublicAndPrivateKeys(2, 2)

			runPeriodicalRotate(nextRotation)

			keys := getJWKS()
			diff := deep.Equal(pubKeys, keys)
			require.Nil(t, diff)

			privKeysNow := getPrivateKeys()
			require.Len(t, privKeysNow, 1)
		})

		t.Run("should stay as is after rotation now", func(t *testing.T) {
			pubKeys, privKeys := assertLenPublicAndPrivateKeys(2, 1)

			runPeriodicalRotate(afterNextRotation)

			assertEqualsKeysAfterRotate(pubKeys, privKeys)
		})
	})

	// first generation after enabling is special
	// but second and next rotation as will same result
	// check it next times

	for j := 1; j <= 3; j++ {
		t.Run(fmt.Sprintf("next cycles after first rotation: iter %v", j), func(t *testing.T) {
			_, genNew, afterGenNew, nextRotation, afterNextRotation := creteIntervals(curRotationTime)
			curRotationTime = nextRotation

			t.Run("should generate new keys and not delete public key on gen time", func(t *testing.T) {
				pub, priv := assertLenPublicAndPrivateKeys(2, 1)

				runPeriodicalRotate(genNew)

				nowPub, nowPriv := assertLenPublicAndPrivateKeys(3, 2)

				for i := 0; i < 2; i++ {
					diff := deep.Equal(pub[i], nowPub[i])
					require.Nil(t, diff)
				}

				diff := deep.Equal(priv[0], nowPriv[0])
				require.Nil(t, diff)
			})

			t.Run("should stay as is after gen time", func(t *testing.T) {
				pub, priv := assertLenPublicAndPrivateKeys(3, 2)

				runPeriodicalRotate(afterGenNew)

				assertEqualsKeysAfterRotate(pub, priv)
			})

			t.Run("should remove priv key and old pub key on next rotation time", func(t *testing.T) {
				pub, priv := assertLenPublicAndPrivateKeys(3, 2)

				runPeriodicalRotate(nextRotation)

				nowPub, nowPriv := assertLenPublicAndPrivateKeys(2, 1)

				for i := 0; i < 2; i++ {
					diff := deep.Equal(pub[i+1], nowPub[i])
					require.Nil(t, diff)
				}

				diff := deep.Equal(priv[1], nowPriv[0])
				require.Nil(t, diff)
			})

			t.Run("should stay as is after rotation time", func(t *testing.T) {
				pub, priv := assertLenPublicAndPrivateKeys(2, 1)

				runPeriodicalRotate(afterNextRotation)

				assertEqualsKeysAfterRotate(pub, priv)
			})
		})
	}
}
