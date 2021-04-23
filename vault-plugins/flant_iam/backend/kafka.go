package backend

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

// -------------------------------- TO EXTRACT TO THE KAFKA LIB ------------------------------ //
type Topic string

const (
	Vault    Topic = "vault"
	Metadata Topic = "metadata"
)

type EntityMarshaller interface {
	Key() string
	Marshal() ([]byte, error)
	PublicMarshal() ([]byte, error)
}

type KafkaSender interface {
	Send(ctx context.Context, marshaller EntityMarshaller, topics []Topic) error
	Delete(ctx context.Context, marshaller EntityMarshaller, topics []Topic) error
}

// ----------------------------- End of TO EXTRACT TO THE KAFKA LIB ------------------------------ //

// Message is used to send the data outside
type Message struct {
	Meta Meta `json:"meta"`
	Data Data `json:"data"`
}

func (m *Message) Key() string {
	return m.Meta.Key
}

func (m *Message) Marshal() ([]byte, error) {
	return jsonutil.EncodeJSON(m)
}

func (m *Message) PublicMarshal() ([]byte, error) {
	safe := &Message{
		Meta: m.Meta,
		Data: m.Data.Clean(),
	}
	return safe.Marshal()
}

type Data interface {
	json.Marshaler
	json.Unmarshaler
	Clean() Data
}

type Meta struct {
	Type string `json:"type"`
	Id   string `json:"id"`
	Key  string `json:"key"`
}
