package jwt

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"golang.org/x/crypto/ed25519"
	"gopkg.in/square/go-jose.v2"
)

func generateKeys() (*jose.JSONWebKey, *jose.JSONWebKey, error) {
	pubKey, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("gen ecdsa key: %v", err)
	}

	priv := jose.JSONWebKey{
		Key:       key,
		KeyID:     newUUID(),
		Algorithm: string(jose.EdDSA),
		Use:       "sig",
	}
	pub := jose.JSONWebKey{
		Key:       pubKey,
		KeyID:     newUUID(),
		Algorithm: string(jose.EdDSA),
		Use:       "sig",
	}

	return &priv, &pub, nil
}

func rotationTimestamp(ctx context.Context, storage logical.Storage, now func() time.Time) error {
	timeNow := now().Unix()
	rotateTime, err := logical.StorageEntryJSON("jwt/keys_last_rotation", timeNow)
	if err != nil {
		return err
	}
	err = storage.Put(ctx, rotateTime)
	if err != nil {
		return err
	}

	return nil
}

func modifyKeys(ctx context.Context, storage logical.Storage, modify func(*jose.JSONWebKeySet, *jose.JSONWebKeySet) error) error {
	entry, err := storage.Get(ctx, "jwt/jwks")
	if err != nil {
		return err
	}

	pubicKeySet := jose.JSONWebKeySet{}
	if entry != nil {
		err = entry.DecodeJSON(&pubicKeySet)
		if err != nil {
			return err
		}
	}

	entry, err = storage.Get(ctx, "jwt/private_keys")
	if err != nil {
		return err
	}

	privateSet := jose.JSONWebKeySet{}
	if entry != nil {
		err = entry.DecodeJSON(&privateSet)
		if err != nil {
			return err
		}
	}

	err = modify(&privateSet, &pubicKeySet)
	if err != nil {
		return err
	}

	pubEntry, err := logical.StorageEntryJSON("jwt/jwks", pubicKeySet)
	if err != nil {
		return err
	}

	err = storage.Put(ctx, pubEntry)
	if err != nil {
		return err
	}

	privEntry, err := logical.StorageEntryJSON("jwt/private_keys", privateSet)
	if err != nil {
		return err
	}
	err = storage.Put(ctx, privEntry)
	if err != nil {
		return err
	}

	return nil
}

// generateOrRotateKeys generates a new keypair and adds it to keys in the storage
func generateOrRotateKeys(ctx context.Context, storage logical.Storage) error {
	return modifyKeys(ctx, storage, func(privateSet, pubicKeySet *jose.JSONWebKeySet) error {
		priv, pub, err := generateKeys()
		if err != nil {
			return err
		}

		privateSet.Keys = append(privateSet.Keys, *priv)
		if len(privateSet.Keys) > 2 {
			privateSet.Keys = privateSet.Keys[1:len(privateSet.Keys)]
		}
		pubicKeySet.Keys = append(pubicKeySet.Keys, *pub)
		if len(pubicKeySet.Keys) > 2 {
			pubicKeySet.Keys = pubicKeySet.Keys[1:len(pubicKeySet.Keys)]
		}

		return nil
	})
}

// removeFirstKey remove the key if there are more than one
func removeFirstKey(ctx context.Context, storage logical.Storage) error {
	return modifyKeys(ctx, storage, func(privateSet, pubicKeySet *jose.JSONWebKeySet) error {
		if len(privateSet.Keys) == 2 {
			privateSet.Keys = privateSet.Keys[1:]
		}
		if len(pubicKeySet.Keys) == 2 {
			pubicKeySet.Keys = pubicKeySet.Keys[1:]
		}
		return nil
	})
}

func PathJWKS(b *TokenController) *framework.Path {
	return &framework.Path{
		Pattern: `jwks`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: protectNonEnabled(b.handleJWKSRead),
				Summary:  pathJWTJWKSSynopsis,
			},
		},

		HelpSynopsis: pathJWTJWKSSynopsis,
	}
}

func (b *TokenController) handleJWKSRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	entry, err := req.Storage.Get(ctx, "jwt/jwks")
	if err != nil {
		return nil, err
	}

	keys := make([]byte, 0)
	if entry != nil {
		keys = entry.Value
	}

	keysSet := jose.JSONWebKeySet{}
	if len(keys) > 0 {
		if err := json.Unmarshal(keys, &keysSet); err != nil {
			return nil, err
		}
	}

	entry, err = req.Storage.Get(ctx, "jwt/external_jwks")
	if err != nil {
		return nil, err
	}

	externalKeys := make([]byte, 0)
	if entry != nil {
		externalKeys = entry.Value
	}

	externalKeysSet := jose.JSONWebKeySet{}
	if len(externalKeys) > 0 {
		if err := json.Unmarshal(externalKeys, &externalKeysSet); err != nil {
			return nil, err
		}
	}

	keysSet.Keys = append(keysSet.Keys, externalKeysSet.Keys...)

	resp := &logical.Response{
		Data: map[string]interface{}{"keys": keysSet.Keys},
	}

	return resp, nil
}

func PathRotateKey(b *TokenController) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/rotate_key`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: protectNonEnabled(b.handleRotateKeysUpdate),
				Summary:  pathJWTRotateKeySynopsis,
			},
		},

		HelpSynopsis: pathJWTRotateKeySynopsis,
	}
}

func (b *TokenController) handleRotateKeysUpdate(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	priv, pub, err := generateKeys()
	if err != nil {
		return nil, err
	}

	privEntry, err := logical.StorageEntryJSON("jwt/private_keys", jose.JSONWebKeySet{Keys: []jose.JSONWebKey{*priv}})
	if err != nil {
		return nil, err
	}
	err = req.Storage.Put(ctx, privEntry)
	if err != nil {
		return nil, err
	}

	pubEntry, err := logical.StorageEntryJSON("jwt/jwks", jose.JSONWebKeySet{Keys: []jose.JSONWebKey{*pub}})
	if err != nil {
		return nil, err
	}
	err = req.Storage.Put(ctx, pubEntry)
	if err != nil {
		return nil, err
	}

	err = rotationTimestamp(ctx, req.Storage, b.now)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"rotated": true},
	}

	return resp, nil
}

const (
	pathJWTJWKSSynopsis = `
Endpoint to expose public keys to check authority of issued tokens.
`
	pathJWTRotateKeySynopsis = `
Force key rotation. Calling this endpoint will rotate keys immediately.
`
)
