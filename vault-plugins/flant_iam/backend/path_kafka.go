package backend

import (
	"context"
	"encoding/json"
	"net/http"

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
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Summary:  "Setup kafka configuration",
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

	if len(kb.broker.GetEndpoints()) == 0 {
		rr := logical.ErrorResponse("kafka connection is not configured. Run /kafka/configure_access first")
		return logical.RespondWithStatusCode(rr, req, http.StatusPreconditionFailed)
	}

	kb.broker.PluginConfig.SelfTopicName = topicName.(string)
	err := kb.broker.CreateTopic(kb.broker.PluginConfig.SelfTopicName) // TODO: ctx
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
