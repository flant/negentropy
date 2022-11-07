package io

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

func Test_storeLastOffset(t *testing.T) {
	ctx := context.TODO()
	storage := &logical.InmemStorage{}
	pt := "plugin-self-topic-name"
	tp := "topic-name"
	expected := int64(123)

	err := StoreLastOffsetToStorage(ctx, storage, pt, tp, 123)

	require.NoError(t, err)
	se, err := storage.Get(ctx, lastOffsetStorageKey(pt, tp))
	require.NoError(t, err)
	require.NotNil(t, se)
	require.Len(t, se.Value, 8)
	actual := int64(binary.BigEndian.Uint64(se.Value))
	require.Equal(t, expected, actual)
}

func Test_getLastOffsetFromStorage(t *testing.T) {
	ctx := context.TODO()
	storage := &logical.InmemStorage{}
	pt := "plugin-self-topic-name"
	tp := "topic-name"
	expected := int64(123)
	err := StoreLastOffsetToStorage(ctx, storage, pt, tp, 123)
	require.NoError(t, err)

	actual, err := LastOffsetFromStorage(ctx, storage, pt, tp)

	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func Test_Test_getLastOffsetFromEmptyStorage(t *testing.T) {
	ctx := context.TODO()
	storage := &logical.InmemStorage{}
	pt := "plugin-self-topic-name"
	tp := "topic-name"

	actual, err := LastOffsetFromStorage(ctx, storage, pt, tp)

	require.NoError(t, err)
	require.Equal(t, int64(-1), actual)
}
