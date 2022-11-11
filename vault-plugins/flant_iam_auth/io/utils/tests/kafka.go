package tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func CreateDecryptCreateMessage(t *testing.T, obj io.MemoryStorableObject) *io.MsgDecoded {
	data, err := json.Marshal(obj)
	require.NoError(t, err)

	return &io.MsgDecoded{
		Type: obj.ObjType(),
		ID:   obj.ObjId(),
		Data: data,
	}
}

func CreateDecryptDeleteMessage(obj io.MemoryStorableObject) *io.MsgDecoded {
	return &io.MsgDecoded{
		Type: obj.ObjType(),
		ID:   obj.ObjId(),
		Data: make([]byte, 0),
	}
}
