package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/flant/negentropy/kafka-consumer/pkg"
	"github.com/flant/negentropy/rolebinding-watcher/internal"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

// environments variables to pass params
const kafkaEndpoints = "KAFKA_ENDPOINTS"                           // example: http://localhost:9094
const kafkaUseSSL = "KAFKA_USE_SSL"                                // example: true
const kafkaCaPath = "KAFKA_CA_PATH"                                // example: /Users/admin/flant/negentropy/docker/kafka/ca.crt
const kafkaPrivateKeyPath = "KAFKA_PRIVATE_KEY_PATH"               // example: /Users/admin/flant/negentropy/docker/kafka/client.key
const kafkaPrivateCertPath = "KAFKA_PRIVATE_CERT_PATH"             // example: /Users/admin/flant/negentropy/docker/kafka/client.crt
const clientTopic = "CLIENT_TOPIC"                                 // example: root_source.foobar
const clientGroupID = "CLIENT_GROUP_ID"                            // example: foobar
const clientEncryptionPrivateKey = "CLIENT_ENCRYPTION_PRIVATE_KEY" // example: "-----BEGIN RSA PRIVATE KEY-----\n ..." it is a private part of key passed to iam to register replica
const clientEncryptionPublicKey = "CLIENT_ENCRYPTION_PUBLIC_KEY"   // example: "-----BEGIN RSA PUBLIC KEY-----\n ..." it is a public key from root-vault iam
const httpUrl = "HTTP_URL"                                         // example: localhost:9200/foobar

func main() {
	viper.SetDefault("author", "https://www.flant.com")
	logger := hclog.Default()
	logger.SetLevel(hclog.Info)

	exec := func(cmd *cobra.Command, args []string) error {
		kafkaCFG, err := collectKafkaBrokerCFG()
		if err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("collected kafka cfg: %#v", kafkaCFG))
		kafkaTopic := os.Getenv(clientTopic)
		logger.Info(fmt.Sprintf("Topic to reading: %s", kafkaTopic))
		clientGroupID := os.Getenv(clientGroupID)
		logger.Info(fmt.Sprintf("GroupID: %s", clientGroupID))

		daemon, err := internal.NewDaemon(*kafkaCFG, kafkaTopic, clientGroupID, logger)

		httpURL := os.Getenv(httpUrl)
		logger.Info(fmt.Sprintf("http gate url: %s", httpURL))
		var procceder internal.UserEffectiveRoleProcessor
		if httpURL == "" {
			procceder = internal.MockProceeder{Logger: logger}
		} else {
			procceder = internal.NewHTTPClient(httpURL)
		}
		return daemon.Run(procceder)
	}

	rootCmd := &cobra.Command{
		Use:   "watcher",
		Short: "Flant negentropy rolebinding-watcher",
		Long: `Flant integration negentropy rolebinding-watcher
	Configure run by passing environment variables:
KAFKA_ENDPOINTS                               // example: localhost:9094
KAFKA_USE_SSL                               // bool
KAFKA_CA_PATH                               // example: /Users/admin/flant/negentropy/docker/kafka/ca.crt
KAFKA_PRIVATE_KEY_PATH                      // example: /Users/admin/flant/negentropy/docker/kafka/client.key
KAFKA_PRIVATE_CERT_PATH                     // example: /Users/admin/flant/negentropy/docker/kafka/client.crt
CLIENT_TOPIC                                // example: root_source.bush
CLIENT_GROUP_ID                             // example: bush
CLIENT_ENCRYPTION_PRIVATE_KEY               // example: "-----BEGIN RSA PRIVATE KEY-----\n ..." it is a private part of key passed to iam to register replica
CLIENT_ENCRYPTION_PUBLIC_KEY"               // example: "-----BEGIN RSA PUBLIC KEY-----\n ..." it is a public key from root-vault iam
HTTP_URL										// example: "localhost:9200/foobar

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
	endpoints := os.Getenv(kafkaEndpoints)
	useSSLraw := os.Getenv(kafkaUseSSL)
	var useSSL bool
	if useSSLraw == "true" {
		useSSL = true
	}
	clientEncryptionPrivateKey, err := pkg.ParseRSAPrivateKey(os.Getenv(clientEncryptionPrivateKey))
	if err != nil {
		return nil, err
	}

	clientEncryptionPublicKey, err := pkg.ParseRSAPubKey(os.Getenv(clientEncryptionPublicKey))
	if err != nil {
		return nil, err
	}

	return &sharedkafka.BrokerConfig{
		Endpoints: strings.Split(endpoints, ","),
		SSLConfig: &sharedkafka.SSLConfig{
			UseSSL:                useSSL,
			CAPath:                os.Getenv(kafkaCaPath),
			ClientPrivateKeyPath:  os.Getenv(kafkaPrivateKeyPath),
			ClientCertificatePath: os.Getenv(kafkaPrivateCertPath),
		},
		EncryptionPrivateKey: clientEncryptionPrivateKey,
		EncryptionPublicKey:  clientEncryptionPublicKey,
	}, nil
}
