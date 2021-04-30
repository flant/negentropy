package jwtauth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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
		},
	}

	return append(bb.broker.KafkaPaths(), configurePath)
}

func (kb kafkaBackend) handleKafkaConfiguration(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if len(kb.broker.GetEndpoints()) == 0 {
		rr := logical.ErrorResponse("kafka connection is not configured. Run /kafka/configure_access first")
		return logical.RespondWithStatusCode(rr, req, http.StatusPreconditionFailed)
	}

	rootTopicNameRaw, ok := data.GetOk("root_topic_name")
	if !ok || rootTopicNameRaw == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "root_topic_name required")
	}
	rootTopicName := rootTopicNameRaw.(string)

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

	rootPublicKey := strings.ReplaceAll(rootPublicKeyRaw.(string), "\\n", "\n")

	pubkey, err := parsePubkey(rootPublicKey)
	if err != nil {
		return nil, err
	}

	var peerKeys []*rsa.PublicKey
	peerKeysRaw, ok := data.GetOk("peers_public_keys")
	if ok {
		peerKeysStr := peerKeysRaw.([]string)

		for _, pks := range peerKeysStr {
			pub, err := parsePubkey(pks)
			if err != nil {
				return nil, err
			}
			peerKeys = append(peerKeys, pub)
		}
	}

	if len(kb.broker.GetEndpoints()) == 0 {
		rr := logical.ErrorResponse("kafka is not configured. Run /kafka/configure_access first")
		return logical.RespondWithStatusCode(rr, req, http.StatusPreconditionFailed)
	}

	kb.broker.PluginConfig.SelfTopicName = selfTopicName
	kb.broker.PluginConfig.RootTopicName = rootTopicName
	kb.broker.PluginConfig.RootPublicKey = pubkey
	if len(peerKeys) > 0 {
		kb.broker.PluginConfig.PeersPublicKeys = peerKeys
	}
	kb.broker.PluginConfig.PublishQuotaUsage = data.Get("publish_quota_usage").(bool)

	err = kb.broker.CreateTopic(kb.broker.PluginConfig.SelfTopicName)
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

	return nil, nil
}

func parsePubkey(data string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(data))
	pubkey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return pubkey, nil
}
