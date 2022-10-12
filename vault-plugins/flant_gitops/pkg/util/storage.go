package util

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/vault/sdk/logical"
)

// GetString returns string saved into storage by key. The pair of PutString
func GetString(ctx context.Context, storage logical.Storage, key string) (string, error) {
	entry, err := storage.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("unable to get key %q from storage: %s", key, err.Error())
	}
	if entry == nil {
		return "", nil
	}
	return string(entry.Value), nil
}

// PutString saves string into storage by key. The pair of GetString
func PutString(ctx context.Context, storage logical.Storage, key string, value string) error {
	return storage.Put(ctx, &logical.StorageEntry{
		Key:   key,
		Value: []byte(value),
	})
}

// GetInt64 returns int64 saved into storage by key. The pair of PutInt64
func GetInt64(ctx context.Context, storage logical.Storage, key string) (int64, error) {
	entry, err := storage.Get(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("unable to get key %q from storage: %s", key, err.Error())
	}
	if entry == nil {
		return 0, nil
	}
	return strconv.ParseInt(string(entry.Value), 10, 64)
}

// PutInt64 saves string into storage by key. The pair of GetInt64
func PutInt64(ctx context.Context, storage logical.Storage, key string, value int64) error {
	return storage.Put(ctx, &logical.StorageEntry{
		Key:   key,
		Value: []byte(fmt.Sprintf("%d", value)),
	})
}

// PutStringMap saves map[string]string by key. The pair of GetStringMap
func PutStringMap(ctx context.Context, storage logical.Storage, key string, value map[string]string) error {
	if value == nil {
		return fmt.Errorf("prohibited store nil, use empty map instead")
	}
	d, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return storage.Put(ctx, &logical.StorageEntry{
		Key:   key,
		Value: d,
	})
}

// GetStringMap returns map[string]string saved into storage by key. The pair of PutInt64
func GetStringMap(ctx context.Context, storage logical.Storage, key string) (map[string]string, error) {
	entry, err := storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("unable to get key %q from storage: %s", key, err.Error())
	}
	if entry == nil {
		return map[string]string{}, nil
	}
	var value map[string]string
	err = json.Unmarshal(entry.Value, &value)
	return value, err
}
