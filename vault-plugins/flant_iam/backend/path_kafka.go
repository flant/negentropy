package backend

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

type kafkaBackend struct {
	logical.Backend
	storage *io.MemoryStore
	broker  *kafka.MessageBroker
	logger  hclog.Logger
}

func kafkaPaths(b logical.Backend, storage *io.MemoryStore, parentLogger hclog.Logger) []*framework.Path {
	bb := kafkaBackend{
		Backend: b,
		storage: storage,
		broker:  storage.GetKafkaBroker(),
		logger:  parentLogger.Named("KafkaPath"),
	}

	configurePath := &framework.Path{
		Pattern: "kafka/configure",
		Fields: map[string]*framework.FieldSchema{
			"self_topic_name": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "Kafka topic name for this plugin entities",
			},

			"peers_public_keys": {
				Type:        framework.TypeCommaStringSlice,
				Description: "Vault public keys to check signature in JWKS topic",
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
			logical.ReadOperation: &framework.PathOperation{
				Summary:  "Read kafka plugin configuration",
				Callback: bb.handleKafkaReadConfiguration,
			},
		},
	}

	return append(bb.broker.KafkaPaths(), configurePath)
}

func (kb kafkaBackend) handleKafkaConfiguration(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	kb.logger.Debug("handleKafkaConfiguration started")
	defer kb.logger.Debug("handleKafkaConfiguration exit")
	topicName, ok := data.GetOk("self_topic_name")
	if !ok || topicName == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "self_topic_name required")
	}

	if len(kb.broker.GetEndpoints()) == 0 {
		rr := logical.ErrorResponse("kafka connection is not configured. Run /kafka/configure_access first")
		return logical.RespondWithStatusCode(rr, req, http.StatusPreconditionFailed)
	}

	kb.broker.PluginConfig.SelfTopicName = topicName.(string)
	err := kb.broker.CreateTopic(ctx, kb.broker.PluginConfig.SelfTopicName, nil)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	var peerKeys []*rsa.PublicKey
	peerKeysRaw, ok := data.GetOk("peers_public_keys")
	if ok {
		peerKeysStr := peerKeysRaw.([]string)
		for _, pks := range peerKeysStr {
			pksTrimmed := strings.ReplaceAll(strings.TrimSpace(pks), "\\n", "\n")
			kb.logger.Debug(fmt.Sprintf("Peers pub keys %s", pksTrimmed))
			pub, err := utils.ParsePubkey(pksTrimmed)
			if err != nil {
				return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
			}
			peerKeys = append(peerKeys, pub)
		}
	} else {
		kb.logger.Warn("Not pass one more peersKeys")
	}

	if len(peerKeys) > 0 {
		kb.broker.PluginConfig.PeersPublicKeys = peerKeys
	}

	// Create JWKS topic
	jwksConfig := map[string]string{
		"cleanup.policy": "compact, delete",
		"retention.ms":   "2678400000", // 31 days
	}
	err = kb.broker.CreateTopic(ctx, "jwks", jwksConfig)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	d, err := json.Marshal(kb.broker.PluginConfig)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	err = req.Storage.Put(ctx, &logical.StorageEntry{Key: kafka.PluginConfigPath, Value: d, SealWrap: true})
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	kb.broker.CheckConfig()
	kb.storage.ReinitializeKafka()

	return logical.RespondWithStatusCode(&logical.Response{}, req, http.StatusOK)
}

func (kb kafkaBackend) handleKafkaReadConfiguration(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cfg := kb.broker.PluginConfig
	cfgResp := struct {
		SelfTopicName     string   `json:"self_topic_name,omitempty"`
		RootTopicName     string   `json:"root_topic_name,omitempty"`
		RootPublicKey     string   `json:"root_public_key,omitempty"`
		PeersPublicKeys   []string `json:"peers_public_keys,omitempty"`
		PublishQuotaUsage bool     `json:"publish_quota,omitempty"`
	}{
		SelfTopicName:     cfg.SelfTopicName,
		RootTopicName:     cfg.RootTopicName,
		RootPublicKey:     backentutils.ConvertToPem(cfg.RootPublicKey),
		PeersPublicKeys:   backentutils.ConvertToPems(cfg.PeersPublicKeys),
		PublishQuotaUsage: cfg.PublishQuotaUsage,
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"kafka_configuration": &cfgResp,
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
