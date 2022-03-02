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

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

type kafkaBackend struct {
	logical.Backend
	storage *sharedio.MemoryStore
	broker  *kafka.MessageBroker
	logger  hclog.Logger
}

func kafkaPaths(b logical.Backend, storage *sharedio.MemoryStore, parentLogger hclog.Logger) []*framework.Path {
	bb := kafkaBackend{
		Backend: b,
		storage: storage,
		broker:  storage.GetKafkaBroker(),
		logger:  parentLogger.Named("KafkaBackend"),
	}

	configurePath := &framework.Path{
		Pattern: "kafka/configure",
		Fields: map[string]*framework.FieldSchema{
			"root_public_key": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "RootVault public key for replication",
			},
			"root_topic_name": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "RootVault replication topic name",
			},
			"self_topic_name": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "Self restore topic name",
			},
			"peers_public_keys": {
				Type:        framework.TypeCommaStringSlice,
				Description: "Vault public keys to check signature in JWKS and TokenGenerationNumber topics",
			},
			"publish_quota_usage": {
				Type:    framework.TypeBool,
				Default: false,
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
	if len(kb.broker.GetEndpoints()) == 0 {
		rr := logical.ErrorResponse("kafka connection is not configured. Run /kafka/configure_access first")
		return logical.RespondWithStatusCode(rr, req, http.StatusPreconditionFailed)
	}

	rootTopicNameRaw, ok := data.GetOk("root_topic_name")
	if !ok || rootTopicNameRaw == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "root_topic_name required")
	}
	rootTopicName := rootTopicNameRaw.(string)

	topicExists, err := kb.broker.TopicExists(rootTopicName)
	if err != nil {
		rr := logical.ErrorResponse(fmt.Sprintf("checking topic %s existence: %s", rootTopicName, err.Error()))
		return logical.RespondWithStatusCode(rr, req, http.StatusInternalServerError)
	}

	if !topicExists {
		rr := logical.ErrorResponse(fmt.Sprintf("topic %s is not exit. Run flant_iam/replica first", rootTopicName))
		return logical.RespondWithStatusCode(rr, req, http.StatusPreconditionFailed)
	}

	splittedTopic := strings.Split(rootTopicName, ".")
	if len(splittedTopic) != 2 {
		return nil, logical.CodedError(http.StatusBadRequest, "root_topic_name must be in form root_source.$replicaName")
	}

	selfTopicNameRaw, ok := data.GetOk("self_topic_name")
	if !ok || rootTopicNameRaw == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "self_topic_name required")
	}
	selfTopicName := selfTopicNameRaw.(string)

	rootPublicKeyRaw, ok := data.GetOk("root_public_key")
	if !ok || rootPublicKeyRaw == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "root_public_key required")
	}

	rootPublicKey := strings.ReplaceAll(strings.TrimSpace(rootPublicKeyRaw.(string)), "\\n", "\n")
	kb.logger.Debug(fmt.Sprintf("Root pub key pub key %s", rootPublicKey))

	pubkey, err := utils.ParsePubkey(rootPublicKey)
	if err != nil {
		return nil, err
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
				return nil, err
			}
			peerKeys = append(peerKeys, pub)
		}
	} else {
		kb.logger.Warn("Not pass one more peersKeys")
	}

	kb.broker.PluginConfig.SelfTopicName = selfTopicName
	kb.broker.PluginConfig.RootTopicName = rootTopicName
	kb.broker.PluginConfig.RootPublicKey = pubkey
	if len(peerKeys) > 0 {
		kb.broker.PluginConfig.PeersPublicKeys = peerKeys
	}
	kb.broker.PluginConfig.PublishQuotaUsage = data.Get("publish_quota_usage").(bool)

	err = kb.broker.CreateTopic(ctx, kb.broker.PluginConfig.SelfTopicName, nil)
	if err != nil {
		return nil, err
	}

	// Create multipass generation number topic
	multipassGenNumberConfig := map[string]string{
		"cleanup.policy": "compact, delete",
		"retention.ms":   "2678400000", // 31 days
	}
	err = kb.broker.CreateTopic(ctx, io.MultipassNumberGenerationTopic, multipassGenNumberConfig)
	if err != nil {
		return nil, err
	}

	d, err := json.Marshal(kb.broker.PluginConfig)
	if err != nil {
		return nil, err
	}

	kb.broker.CheckConfig()
	err = req.Storage.Put(ctx, &logical.StorageEntry{Key: kafka.PluginConfigPath, Value: d, SealWrap: true})
	if err != nil {
		return nil, err
	}

	kb.storage.ReinitializeKafka()

	return &logical.Response{}, nil
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
