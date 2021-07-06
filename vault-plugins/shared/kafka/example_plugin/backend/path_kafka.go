package backend

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type kafkaBackend struct {
	logical.Backend
	storage *io.MemoryStore
	broker  *kafka.MessageBroker
}

func kafkaPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := kafkaBackend{
		Backend: b,
		storage: storage,
		broker:  storage.GetKafkaBroker(),
	}

	configurePath := &framework.Path{
		Pattern: "kafka/configure",
		Fields: map[string]*framework.FieldSchema{
			"self_topic_name": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "Kafka topic name for this plugin entities",
			},
			"root_public_key": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "RSA key to encrypt messages for destination",
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Summary:  "Setup kafka plugin configuration",
				Callback: bb.handleKafkaConfiguration,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Summary:  "Setup kafka plugin configuration",
				Callback: bb.handleKafkaConfiguration,
			},
		},
	}

	return append(bb.broker.KafkaPaths(), configurePath)
}

func (kb kafkaBackend) handleKafkaConfiguration(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	topicName, ok := data.GetOk("self_topic_name")
	if !ok || topicName == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "self_topic_name required")
	}

	rootPublicKeyRaw, ok := data.GetOk("root_public_key")
	if !ok || rootPublicKeyRaw == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "root_public_key required")
	}

	rootPublicKey := strings.ReplaceAll(strings.TrimSpace(rootPublicKeyRaw.(string)), "\\n", "\n")

	block, _ := pem.Decode([]byte(rootPublicKey))
	if block == nil {
		return nil, errors.New("key can not be parsed")
	}
	pubkey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	if len(kb.broker.GetEndpoints()) == 0 {
		rr := logical.ErrorResponse("kafka connection is not configured. Run /kafka/configure_access first")
		return logical.RespondWithStatusCode(rr, req, http.StatusPreconditionFailed)
	}

	kb.broker.PluginConfig.SelfTopicName = topicName.(string)
	kb.broker.PluginConfig.RootPublicKey = pubkey
	err = kb.broker.CreateTopic(ctx, kb.broker.PluginConfig.SelfTopicName)
	if err != nil {
		return nil, err
	}

	d, err := json.Marshal(kb.broker.PluginConfig)
	if err != nil {
		return nil, err
	}

	err = req.Storage.Put(ctx, &logical.StorageEntry{Key: kafka.PluginConfigPath, Value: d, SealWrap: true})
	if err != nil {
		return nil, err
	}

	kb.broker.CheckConfig()
	kb.storage.ReinitializeKafka()

	return &logical.Response{}, nil
}
