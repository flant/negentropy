package kafka

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	kafkaConfigPath  = "kafka.config"
	PluginConfigPath = "kafka.plugin.config"
)

func (mb *MessageBroker) handlePublicKeyRead(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	if mb.KafkaConfig.EncryptionPublicKey == nil {
		return nil, logical.CodedError(http.StatusNotFound, "public key does not exist. Run /kafka/configure_access first")
	}
	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(mb.KafkaConfig.EncryptionPublicKey),
		},
	)

	return &logical.Response{
		Data: map[string]interface{}{
			"public_key": strings.ReplaceAll(string(pemdata), "\n", "\\n"),
		},
	}, nil
}

func (mb *MessageBroker) handleConfigureAccess(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// kafka backends
	endpoints := data.Get("kafka_endpoints").([]string)
	if len(endpoints) == 0 {
		return nil, logical.CodedError(http.StatusBadRequest, "endpoints required")
	}
	mb.KafkaConfig.Endpoints = endpoints
	sslConfig, err := parseSSLCfg(data)
	if err != nil {
		mb.Logger.Error(err.Error())
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}
	mb.KafkaConfig.SSLConfig = sslConfig
	// TODO: check kafka connection
	// generate encryption keys
	if mb.KafkaConfig.EncryptionPrivateKey == nil {
		pk, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
		}
		mb.KafkaConfig.EncryptionPrivateKey = pk
		mb.KafkaConfig.EncryptionPublicKey = &pk.PublicKey
	}

	d, err := json.Marshal(mb.KafkaConfig)
	if err != nil {
		return nil, err
	}

	err = req.Storage.Put(ctx, &logical.StorageEntry{Key: kafkaConfigPath, Value: d, SealWrap: true})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// parseSSLCfg parse and check ssl config
func parseSSLCfg(data *framework.FieldData) (*SSLConfig, error) {
	cfg := SSLConfig{
		UseSSL:                data.Get("use_ssl").(bool),
		CAPath:                data.Get("ca_path").(string),
		ClientPrivateKeyPath:  data.Get("client_private_key_path").(string),
		ClientCertificatePath: data.Get("client_certificate_path").(string),
	}
	if !cfg.UseSSL {
		return nil, nil
	}
	if cfg.UseSSL && (cfg.CAPath == "" || cfg.ClientCertificatePath == "" || cfg.ClientPrivateKeyPath == "") {
		return nil, fmt.Errorf("if passed 'use_ssl=true', have to pass ca_path, client_private_key_path and client_certificate_path")
	}
	return &cfg, nil
}

func (mb *MessageBroker) KafkaPaths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "kafka/public_key",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Summary:  "Return public key",
					Callback: mb.handlePublicKeyRead,
				},
			},
		},
		{
			Pattern: "kafka/configure_access",
			Fields: map[string]*framework.FieldSchema{
				"kafka_endpoints": {
					Type:        framework.TypeStringSlice,
					Required:    true,
					Description: "List of kafka backends. Ex: 192.168.1.1:9093",
				},
				"use_ssl": {
					Type:        framework.TypeBool,
					Required:    true,
					Description: "Use SSL or not, if set true, have to pass ca_path, client_private_key_path, client_certificate_path",
				},
				// TODO more info about ca_path, client_private_key_path,  client_certificate_path
				"ca_path": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "Absolute path to file, which contains CA certificate for verifying the kafka broker key, example: /etc/ca.crt",
				},
				"client_private_key_path": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "Absolute path to file, which contains client private key (PEM) used for authentication, example: /etc/client.key",
				},
				"client_certificate_path": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "Absolute path to file, which contains client public key (PEM) used for authentication, example: /etc/client.crt",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Summary:  "Setup kafka configuration",
					Callback: mb.handleConfigureAccess,
				},
			},
		},
	}
}
