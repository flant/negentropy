package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/flant/negentropy/kafka-consumer/internal"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const serverKafka = "localhost:9094"

// environments variables to pass params
const kafkaCfgServers = "KAFKA_CFG_SERVERS"                               // example: http://localhost:9094
const kafkaCfgUseSSL = "KAFKA_CFG_USE_SSL"                                // example: true
const kafkaCfgCaPath = "KAFKA_CFG_CA_PATH"                                // example: /Users/admin/flant/negentropy/docker/kafka/ca.crt
const kafkaCfgPrivateKeyPath = "KAFKA_CFG_PRIVATE_KEY_PATH"               // example: /Users/admin/flant/negentropy/docker/kafka/client.key
const kafkaCfgPrivateCertPath = "KAFKA_CFG_PRIVATE_CERT_PATH"             // example: /Users/admin/flant/negentropy/docker/kafka/client.crt
const clientCfgTopic = "CLIENT_CFG_TOPIC"                                 // example: root_source.bush
const clientCfgGroupID = "CLIENT_CFG_GROUP_ID"                            // example: bush
const clientCfgEncryptionPrivateKey = "CLIENT_CFG_ENCRYPTION_PRIVATE_KEY" // example: "-----BEGIN RSA PRIVATE KEY-----\n ..." it is a private part of key passed to iam to register replica
const clientCfgEncryptionPublicKey = "CLIENT_CFG_ENCRYPTION_PUBLIC_KEY"   // example: "-----BEGIN RSA PUBLIC KEY-----\n ..." it is a public key from root-vault iam
const httpUrl = "HTTP_URL"                                                // example: localhost:9200/asdf

func main() {
	viper.SetDefault("author", "https://www.flant.com")
	logger := hclog.Default()
	logger.SetLevel(hclog.Debug)

	exec := func(cmd *cobra.Command, args []string) error {
		kafkaCFG, err := collectKafkaBrokerCFG()
		if err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("collected kafka cfg: %#v", kafkaCFG))
		kafkaTopic := os.Getenv(clientCfgTopic)
		logger.Info(fmt.Sprintf("Topic to reading: %s", kafkaTopic))
		clientGroupID := os.Getenv(clientCfgGroupID)
		logger.Info(fmt.Sprintf("GroupID: %s", clientGroupID))
		httpURL := os.Getenv(httpUrl)
		logger.Info(fmt.Sprintf("http gate url: %s", httpURL))

		kfs, err := internal.NewKafkaSource(
			*kafkaCFG,
			kafkaTopic,
			clientGroupID,
			logger,
			internal.NewHTTPClient(httpURL),
		)
		if err != nil {
			return err
		}
		kfs.Run()
		return nil
	}

	rootCmd := &cobra.Command{
		Use:   "consumer",
		Short: "Flant negentropy kafka-consumer",
		Long: `Flant integration kafka-consumer
	Configure run by passing environment variables:
KAFKA_CFG_SERVERS                               // example: localhost:9094
KAFKA_CFG_USE_SSL                               // bool
KAFKA_CFG_CA_PATH                               // example: /Users/admin/flant/negentropy/docker/kafka/ca.crt
KAFKA_CFG_PRIVATE_KEY_PATH                      // example: /Users/admin/flant/negentropy/docker/kafka/client.key
KAFKA_CFG_PRIVATE_CERT_PATH                     // example: /Users/admin/flant/negentropy/docker/kafka/client.crt
CLIENT_CFG_TOPIC                                // example: root_source.bush
CLIENT_CFG_GROUP_ID                             // example: bush
CLIENT_CFG_ENCRYPTION_PRIVATE_KEY               // example: "-----BEGIN RSA PRIVATE KEY-----\n ..." it is a private part of key passed to iam to register replica
CLIENT_CFG_ENCRYPTION_PUBLIC_KEY"               // example: "-----BEGIN RSA PUBLIC KEY-----\n ..." it is a public key from root-vault iam
HTTP_URL										// example: "localhost:9200/asdf

	Find more information at https://flant.com
`,
		RunE: exec,
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func collectKafkaBrokerCFG() (*sharedkafka.BrokerConfig, error) {
	endpoints := os.Getenv(kafkaCfgServers)
	useSSLraw := os.Getenv(kafkaCfgUseSSL)
	var useSSL bool
	if useSSLraw == "true" {
		useSSL = true
	}
	clientEncryptionPrivateKey, err := internal.ParseRSAPrivateKey(os.Getenv(clientCfgEncryptionPrivateKey))
	if err != nil {
		return nil, err
	}

	clientEncryptionPublicKey, err := internal.ParseRSAPubKey(os.Getenv(clientCfgEncryptionPublicKey))
	if err != nil {
		return nil, err
	}

	return &sharedkafka.BrokerConfig{
		Endpoints: strings.Split(endpoints, ","),
		SSLConfig: &sharedkafka.SSLConfig{
			UseSSL:                useSSL,
			CAPath:                os.Getenv(kafkaCfgCaPath),
			ClientPrivateKeyPath:  os.Getenv(kafkaCfgPrivateKeyPath),
			ClientCertificatePath: os.Getenv(kafkaCfgPrivateCertPath),
		},
		EncryptionPrivateKey: clientEncryptionPrivateKey,
		EncryptionPublicKey:  clientEncryptionPublicKey,
	}, nil
}
