package backend

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_destination"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func TestT(t *testing.T) {
	rs, _ := rsa.GenerateKey(rand.Reader, 4096)

	ss := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&rs.PublicKey),
	})
	pk := strings.ReplaceAll(string(ss), "\n", "\\n")
	fmt.Printf("vault write flant_iam/replica/myrep type=Vault public_key=\"%s\"\n", pk)
	// -----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEArCktBYYir7OuqNQJ4VcOrxmuB8GVWofwerrZres9jzVUvvip\nX++ycjbJKIT8Oi5zKO8TdyDxEj0r9l4xErzRi4wdEgXTubmDhMPAp4+5RYGH6zCr\nVDzHx0LIY7rMWFE1MbTedGwvUmSeKZKPGYusceciV/lhVcqAB+Cnh8iOYQOurc0Q\nqf8vOy/iJivZ6Cosz7CkxUCRMg10u5mb1VY/bWeVl/Rsvsm8o1+vMQKupZY4a79H\n0WJSk9uwcV41cN6EQwHb987qZistmE4uXltPqHoVzkbkYlERQbSHMI7qrNwN1XdD\npJT+CxD6YXOwFOjWsxIgQ6spGMPm7boB71zI9wIDAQABAoIBAEsWinRmVKqdjAhG\nsyh9eAIXCTiIzkN2FwTwihC5EVhswlGo0vbs7L+z9XieyAP4TnIEFFFZJMv3sjz6\nSB0MDbj3m5ZIxFe0+g/l8RkkLoKKRGXoDFHpUJkwH4af6pB6muDbKktNBDbDe9hV\n++QAb24eiXQlaLaqY70L1wX6C190LItnTp8beYNC673odvAyqPLx9hD65zkUGAYK\nK0gidkleJ7uaShlQ823QRFJSjOrb0enyY8bTkViiFE7TwE7QyHYy9Zofu6pfOTXP\nVB0f1vkwVLty0+YlnanPA5NLBB0mJPwy7Bo34Ui2gomx2WGdE4Yq6V5N5T59pWPs\neJtNWYECgYEAzlNj8a0eAy18en3QCfeHR094rRUBQZcWQJK4uhtDEPeAGeAn7btR\nnHRGteidMAr/LPt4DzKmxV9q4bxjemKx54aey2IGKcJ6jm8M4njCjlIpgxHTdB7G\n2KnP+nezSKtd0L2bKqHvqx3eyUUnDENYzevF2/6Q5yqF+F802rq/9skCgYEA1ZwT\n6bIPTuLQt9MqsI1RNPr/f92p7AzJDWD1peE3F21Wmaa6dhjLDVn7XSM4k+gbaIer\nMPFZZEGFio+maH5LWx1wpNoWzCftWPjjNG2M0ysd77O2uOgaV/sDKsIKo7lSqEIi\nGHZ3OPwHddHRO9NeHvXiKNFTQgjeiHGOtqXV4b8CgYEAse9axwbcVjM5Ic40xxOw\nl8Aiu2nc/nrVFvUx2FZAfXZlBGu7I5ujI0Fn5eNpBBDHxjxMaxbsmlTSsUCtrdNF\nx/ziH1Y3KHZvCT0eKIWqi+CxqjaKXJ9aL2orUb8/X5FiQ+3wzlB/h9wn0P4RUdMW\n1+fYaARfZOzYQr2gsG3TtXkCgYEAmKZLDKx0iBlKsrMzRKwYplXglI3hypBwdSEf\nKwXBCvrV8kPV6GNbaBUvrqVm3zv1qkOZsQYorZ6tQhHaB76JN3nYb9ZyiD7YPMbQ\nBz1qb9XWNOAm6gjkGo+E+d9lHw9m5FsuZnDyTkS9SBNDBQ/NqS5qCmVcrEoOTU9p\nf1kPeZ0CgYBsIwl7mBLEoT6kpg7CD/sbveCXC26MtF/lLIsXj7aRbiLgGTNa0h8Q\nEREdzrjZVUqRFgs8BTE4rBsEJ7rs6/KuNdbWaduzczt65C67BO5SF4ehxEPP+pIB\n2X6VkRMUfvkUp031I2Bue+lnRxsFFJugHPfpxUCJmuZJ2BsbG9lIKA==\n-----END RSA PRIVATE KEY-----\n

	// vault write flant_iam/replica/myrep topic_type=Vault public_key="-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEArCktBYYir7OuqNQJ4VcOrxmuB8GVWofwerrZres9jzVUvvip\nX++ycjbJKIT8Oi5zKO8TdyDxEj0r9l4xErzRi4wdEgXTubmDhMPAp4+5RYGH6zCr\nVDzHx0LIY7rMWFE1MbTedGwvUmSeKZKPGYusceciV/lhVcqAB+Cnh8iOYQOurc0Q\nqf8vOy/iJivZ6Cosz7CkxUCRMg10u5mb1VY/bWeVl/Rsvsm8o1+vMQKupZY4a79H\n0WJSk9uwcV41cN6EQwHb987qZistmE4uXltPqHoVzkbkYlERQbSHMI7qrNwN1XdD\npJT+CxD6YXOwFOjWsxIgQ6spGMPm7boB71zI9wIDAQABAoIBAEsWinRmVKqdjAhG\nsyh9eAIXCTiIzkN2FwTwihC5EVhswlGo0vbs7L+z9XieyAP4TnIEFFFZJMv3sjz6\nSB0MDbj3m5ZIxFe0+g/l8RkkLoKKRGXoDFHpUJkwH4af6pB6muDbKktNBDbDe9hV\n++QAb24eiXQlaLaqY70L1wX6C190LItnTp8beYNC673odvAyqPLx9hD65zkUGAYK\nK0gidkleJ7uaShlQ823QRFJSjOrb0enyY8bTkViiFE7TwE7QyHYy9Zofu6pfOTXP\nVB0f1vkwVLty0+YlnanPA5NLBB0mJPwy7Bo34Ui2gomx2WGdE4Yq6V5N5T59pWPs\neJtNWYECgYEAzlNj8a0eAy18en3QCfeHR094rRUBQZcWQJK4uhtDEPeAGeAn7btR\nnHRGteidMAr/LPt4DzKmxV9q4bxjemKx54aey2IGKcJ6jm8M4njCjlIpgxHTdB7G\n2KnP+nezSKtd0L2bKqHvqx3eyUUnDENYzevF2/6Q5yqF+F802rq/9skCgYEA1ZwT\n6bIPTuLQt9MqsI1RNPr/f92p7AzJDWD1peE3F21Wmaa6dhjLDVn7XSM4k+gbaIer\nMPFZZEGFio+maH5LWx1wpNoWzCftWPjjNG2M0ysd77O2uOgaV/sDKsIKo7lSqEIi\nGHZ3OPwHddHRO9NeHvXiKNFTQgjeiHGOtqXV4b8CgYEAse9axwbcVjM5Ic40xxOw\nl8Aiu2nc/nrVFvUx2FZAfXZlBGu7I5ujI0Fn5eNpBBDHxjxMaxbsmlTSsUCtrdNF\nx/ziH1Y3KHZvCT0eKIWqi+CxqjaKXJ9aL2orUb8/X5FiQ+3wzlB/h9wn0P4RUdMW\n1+fYaARfZOzYQr2gsG3TtXkCgYEAmKZLDKx0iBlKsrMzRKwYplXglI3hypBwdSEf\nKwXBCvrV8kPV6GNbaBUvrqVm3zv1qkOZsQYorZ6tQhHaB76JN3nYb9ZyiD7YPMbQ\nBz1qb9XWNOAm6gjkGo+E+d9lHw9m5FsuZnDyTkS9SBNDBQ/NqS5qCmVcrEoOTU9p\nf1kPeZ0CgYBsIwl7mBLEoT6kpg7CD/sbveCXC26MtF/lLIsXj7aRbiLgGTNa0h8Q\nEREdzrjZVUqRFgs8BTE4rBsEJ7rs6/KuNdbWaduzczt65C67BO5SF4ehxEPP+pIB\n2X6VkRMUfvkUp031I2Bue+lnRxsFFJugHPfpxUCJmuZJ2BsbG9lIKA==\n-----END RSA PRIVATE KEY-----\n"
}

func TestReplicas(t *testing.T) {
	t.Skip("not working yet") // TODO: kafka todo
	b, storage := generateBackend(t)

	pk, _ := rsa.GenerateKey(rand.Reader, 4096)
	d := x509.MarshalPKCS1PublicKey(&pk.PublicKey)

	pubS := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: d})
	pub := strings.ReplaceAll(string(pubS), "\n", "\\n")

	t.Run("create replicas", func(t *testing.T) {
		req := &logical.Request{
			Storage:   storage,
			Data:      map[string]interface{}{"type": "Vault", "public_key": pub},
			Path:      "replica/one",
			Operation: logical.UpdateOperation,
		}

		_, err := b.HandleRequest(context.Background(), req)
		require.NoError(t, err)

		req = &logical.Request{
			Storage:   storage,
			Data:      map[string]interface{}{"type": "Metadata", "public_key": pub},
			Path:      "replica/two",
			Operation: logical.UpdateOperation,
		}

		_, err = b.HandleRequest(context.Background(), req)
		require.NoError(t, err)

		req = &logical.Request{
			Storage:   storage,
			Data:      map[string]interface{}{"type": "Vault", "public_key": pub},
			Path:      "replica/three",
			Operation: logical.UpdateOperation,
		}

		_, err = b.HandleRequest(context.Background(), req)
		require.NoError(t, err)
	})

	t.Run("list replicas", func(t *testing.T) {
		req := &logical.Request{
			Storage:   storage,
			Path:      "replica",
			Operation: logical.ReadOperation,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		require.NoError(t, err)
		names := resp.Data["replicas_names"].([]string)
		assert.Len(t, names, 3)
	})

	t.Run("read replica", func(t *testing.T) {
		req := &logical.Request{
			Storage:   storage,
			Path:      "replica/two",
			Operation: logical.ReadOperation,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		require.NoError(t, err)
		data := resp.Data
		assert.Equal(t, "two", data["replica_name"].(string))
		assert.Equal(t, "Metadata", data["type"].(string))
		assert.Equal(t, pub, data["public_key"].(string))
	})

	t.Run("delete replica", func(t *testing.T) {
		req := &logical.Request{
			Storage:   storage,
			Path:      "replica/one",
			Operation: logical.DeleteOperation,
		}

		_, err := b.HandleRequest(context.Background(), req)
		require.NoError(t, err)
	})

	t.Run("read non-existant replica", func(t *testing.T) {
		req := &logical.Request{
			Storage:   storage,
			Path:      "replica/one",
			Operation: logical.ReadOperation,
		}

		res, err := b.HandleRequest(context.Background(), req)
		require.NoError(t, err)
		assert.Contains(t, res.Data["http_raw_body"], "replica not found")
	})
}

func generateBackend(t *testing.T) (logical.Backend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	config := &logical.BackendConfig{
		// Logger: logging.NewVaultLogger(log.Trace),

		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}

	b := &framework.Backend{
		Help:        strings.TrimSpace(commonHelp),
		BackendType: logical.TypeLogical,
	}

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), config.StorageView, log.NewNullLogger())
	if err != nil {
		t.Fatal(err)
	}
	schema, err := iam_repo.GetSchema()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := sharedio.NewMemoryStore(schema, mb)
	if err != nil {
		t.Fatal(err)
	}

	// destinations
	storage.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))

	b.Paths = framework.PathAppend(
		replicasPaths(b, storage),
		tenantPaths(b, storage),
		userPaths(b, nil, storage),
	)
	err = b.Setup(context.TODO(), config)
	if err != nil {
		t.Fatal(err)
	}

	return b, config.StorageView
}
