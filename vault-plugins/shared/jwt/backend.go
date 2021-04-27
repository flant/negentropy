package jwt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// Factory is used by framework
func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := backend()
	if err := b.Setup(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

// Simple backend for test purposes (treat it like an example)
type jwtAuthBackend struct {
	*framework.Backend

	tokenController *TokenController
}

func backend() *jwtAuthBackend {
	b := new(jwtAuthBackend)
	b.tokenController = NewTokenController()

	b.Backend = &framework.Backend{
		BackendType:  logical.TypeCredential,
		Help:         backendHelp,
		PathsSpecial: &logical.Paths{},
		Paths: framework.PathAppend(
			[]*framework.Path{
				PathEnable(b.tokenController),
				PathDisable(b.tokenController),
				PathConfigure(b.tokenController),
				PathJWKS(b.tokenController),
				PathRotateKey(b.tokenController),
			},
		),
		PeriodicFunc: b.tokenController.rotateKeys,
	}

	return b
}

type TokenController struct {
	now func() time.Time
	// mu sync.RWMutex
}

func NewTokenController() *TokenController {
	return &TokenController{now: time.Now}
}

func (b *TokenController) rotateKeys(ctx context.Context, req *logical.Request) error {
	entry, err := req.Storage.Get(ctx, "jwt/enable")
	if err != nil {
		return err
	}

	var enabled bool
	if entry != nil {
		err = entry.DecodeJSON(&enabled)
		if err != nil {
			return err
		}
	}

	if !enabled {
		return nil
	}

	shouldRotate, shouldPublish, err := b.shouldRotateOrPublish(ctx, req.Storage)
	if err != nil {
		return err
	}

	if shouldRotate {
		err := removeFirstKey(ctx, req.Storage)
		if err != nil {
			return err
		}
	} else if shouldPublish {
		err := generateOrRotateKeys(ctx, req.Storage)
		if err != nil {
			return err
		}

		err = rotationTimestamp(ctx, req.Storage, b.now)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *TokenController) shouldRotateOrPublish(ctx context.Context, storage logical.Storage) (bool, bool, error) {
	config, err := getConfig(ctx, storage)
	if err != nil {
		return false, false, err
	}

	rotateEvery, err := time.ParseDuration(config["rotation_period"].(string))
	if err != nil {
		return false, false, err
	}

	publishKeyBefore, err := time.ParseDuration(config["preliminary_announce_period"].(string))
	if err != nil {
		return false, false, err
	}

	lastRotationTime, err := storage.Get(ctx, "jwt/keys_last_rotation")
	if err != nil {
		return false, false, err
	}

	if lastRotationTime == nil {
		return false, false, fmt.Errorf("rotation timestamp not in the store")
	}

	var timeSeconds int64
	err = json.Unmarshal(lastRotationTime.Value, &timeSeconds)
	if err != nil {
		return false, false, err
	}

	lastRotation := time.Unix(timeSeconds, 0)
	now := b.now()

	if !lastRotation.Add(rotateEvery).After(now) {
		return true, false, nil
	} else if !lastRotation.Add(rotateEvery).Add(-publishKeyBefore).After(now) {
		return false, true, nil
	}
	return false, false, nil
}

const (
	backendHelp = `
The JWT backend allows to generate JWT tokens.
`
)
