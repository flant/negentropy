package tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func CreateDecryptCreateMessage(t *testing.T, obj io.MemoryStorableObject) *sharedkafka.MsgDecoded{
	data, err := json.Marshal(obj)
	require.NoError(t, err)

	return &sharedkafka.MsgDecoded{
		Type: obj.ObjType(),
		ID:   obj.ObjId(),
		Data: data,
	}
}

func CreateDecryptDeleteMessage(obj io.MemoryStorableObject) *sharedkafka.MsgDecoded{
	return &sharedkafka.MsgDecoded{
		Type: obj.ObjType(),
		ID:   obj.ObjId(),
		Data: make([]byte, 0),
	}
}
