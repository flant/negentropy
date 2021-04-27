package kafka

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/segmentio/kafka-go"
)

const (
	kafkaConfigPath = "kafka.config"
)

func NewMessageBroker(ctx context.Context, storage logical.Storage, selfHealTopic string) (*MessageBroker, error) {
	if len(selfHealTopic) == 0 {
		return nil, errors.New("topic required")
	}
	mb := &MessageBroker{
		selfHealTopic: selfHealTopic,
	}

	// load encryption private key
	se, err := storage.Get(ctx, kafkaConfigPath)
	if err != nil {
		return nil, err
	}
	if se != nil {
		var config BrokerConfig

		err = json.Unmarshal(se.Value, &config)
		if err != nil {
			return nil, err
		}

		mb.config = config
	}

	if len(mb.config.Endpoints) > 0 &&
		mb.config.EncryptionPublicKey != nil &&
		mb.config.EncryptionPrivateKey != nil {
		mb.isConfigured = true
	}

	return mb, nil
}

func (mb *MessageBroker) handlePublicKeyRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if mb.config.EncryptionPublicKey == nil {
		return nil, logical.CodedError(http.StatusNotFound, "public key does not exist. Run /kafka/configure_access first")
	}
	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(mb.config.EncryptionPublicKey),
		},
	)

	return &logical.Response{
		Data: map[string]interface{}{
			"public_key": strings.Replace(string(pemdata), "\n", "\\n", -1),
		},
	}, nil
}

func (mb *MessageBroker) handleConfigureAccess(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	certData := data.Get("certificate").(string)
	certData = strings.Replace(certData, "\\n", "\n", -1)

	// kafka backends
	endpoints := data.Get("kafka_endpoints").([]string)
	if len(endpoints) == 0 {
		return nil, logical.CodedError(http.StatusBadRequest, "endpoints required")
	}

	// validate certificate
	m, err := x509.MarshalECPrivateKey(mb.config.ConnectionPrivateKey)
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}

	priv := pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: m,
	})

	_, err = tls.X509KeyPair([]byte(certData), priv)
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}

	p, _ := pem.Decode([]byte(certData))
	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}

	mb.config.ConnectionCertificate = cert
	mb.config.Endpoints = endpoints

	// generate encryption keys
	if mb.config.EncryptionPrivateKey == nil {
		pk, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
		}
		mb.config.EncryptionPrivateKey = pk
		mb.config.EncryptionPublicKey = &pk.PublicKey
	}

	d, err := json.Marshal(mb.config)
	if err != nil {
		return nil, err
	}

	t := kafka.TopicConfig{
		Topic: mb.selfHealTopic,
		ConfigEntries: []kafka.ConfigEntry{
			{
				ConfigName:  "cleanup.policy",
				ConfigValue: "compact",
			},
		},
	}
	err = mb.CreateTopic(t)
	if err != nil {
		return nil, err
	}

	err = req.Storage.Put(ctx, &logical.StorageEntry{Key: kafkaConfigPath, Value: d, SealWrap: true})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (mb *MessageBroker) handleGenerateCSR(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	force := data.Get("force").(bool)
	// enforce rotation
	if force {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
		}
		mb.config.ConnectionPrivateKey = priv
	}

	// first run
	var warnings []string
	if mb.config.ConnectionPrivateKey == nil {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
		}
		mb.config.ConnectionPrivateKey = priv
	} else {
		if !force {
			warnings = []string{"Private key is already exist. Add ?force=true param to recreate it"}
		}
	}

	cr := &x509.CertificateRequest{
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		Subject:            pkix.Name{CommonName: "flant_iam.kafka"},
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, cr, mb.config.ConnectionPrivateKey)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	csrr := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: csr,
	})

	return &logical.Response{
		Data: map[string]interface{}{
			"certificate_request": strings.Replace(string(csrr), "\n", "\\n", -1),
		},
		Warnings: warnings,
	}, nil
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
				"certificate": {
					Type:        framework.TypeString,
					Required:    true,
					Description: " x509 certificate to establish Kafka TLS connection",
				},
				"kafka_endpoints": {
					Type:        framework.TypeStringSlice,
					Required:    true,
					Description: "List of kafka backends. Ex: 192.168.1.1:9093",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Summary:  "Setup kafka configuration",
					Callback: mb.handleConfigureAccess,
				},
			},
		},
		{
			Pattern: "kafka/generate_csr",
			Fields: map[string]*framework.FieldSchema{
				"force": {
					Type:        framework.TypeBool,
					Default:     false,
					Description: "Ensforce private key recreation",
					Query:       true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Summary:  "Generate CSR for kafka endpoint",
					Callback: mb.handleGenerateCSR,
				},
			},
		},
	}
}
