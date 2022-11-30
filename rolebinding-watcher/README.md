# rolebinding-watcher

Negentropy rolebinding-watcher daemon watch kafka and produce requests to microservice driven by appearing and disaapearing roles of users

## Build
```
cd negentropy/rolebinding-watcher
GO111MODULE=on go build -o build/rolebinding-watcher main.go
```
## Deployment

- `build/rolebinding-watcher` should be delivered to server or container which have access both to negentropy kafka and microservice
- rolebinding-watcher should be configured:
  * kafka endpoint and topic
  * encryption and decryption rsa keys
  * path to microservice endpoints

## Development

For tests and debug:

- run negentropy environment for e2e tests (from root directory):
```
./start.sh e2e
```
- generate rsa key pair:
```
ssh-keygen -t rsa -m PEM -N '' -f ./id_rsa <<<y >/dev/null 2>&1
ssh-keygen -e -f ./id_rsa -m PEM > ./id_rsa.pub
```
- get token for root vault:
```
cat /tmp/vaults | jq '[.[]|select(.name=="root")][0].token'
```
- create special replica kafka topic for rolebinding-watcher reading:
  * **TOKEN** is a token gotten at previous step (without quotes)
  * **NAME** is a name of replica, topic-name will be `root_source.NAME`
  * **public_key** is a public part of pair, which was generated at early step and save at `./id_rsa.pub`
```
curl --location --request POST 'https://localhost:8300/v1/flant/replica/NAME' \
--header 'X-Vault-Token: TOKEN' \
--header 'Content-Type: application/json' \
--data-raw '{ "type":"Vault", "public_key":"-----BEGIN RSA PUBLIC KEY-----\\nMIIBigKCAYEA4cS4zynvKjYPzVVz921JXWLuElks/cs6CBvJK9UAWdapAg4P+Hb8\\ni2ZycG/r4UEjeffpfBQlwqbE75v29mpxhidE+c6Qs5zJfe5+lyIh0AW+m9TC9IFO\\n6o6NV/Z8foyH+oPzf1ZgKcuTXUc7xlRNK2niun9HJHzrUOLVN1CmBbwu0jyXY+Jq\\n8hl5NYsHLuvGwciyBLERtrIM6bp6a0fLl1ypsloZYW80MyTl7oX6V+sdoQlIIBcJ\\nlCevWMqn9NqhlFSCtL0fdQHJLXOqo6H6WZrEIwWbWGjd0iMTtXIcUPbZ04YUEtCf\\nlsV4YewaoXdANZDJRc798UeBuya8AjWiCt+4/TKdCjlpYmhJ2eCrAhGU0sAFoc81\\nmfJmJb/8OgfwOAzJ8BgGYshukwEXUvQX6V8P5EbTQT97N/rjPQyBFkZh61qv5+MM\\naiIfu2D/wOprDg2mibhehbMV7SarUdVLgIhd8FJ46CsA9riuAR0w0ICe5ndt2M6s\\n80Vn72rBbU47AgMBAAE=\\n-----END RSA PUBLIC KEY-----"}' \
-k
```
- configure rolebinding-watcher by setting environment variables:
  * KAFKA_ENDPOINTS: localhost:9094
  * KAFKA_USE_SSL: true
  * KAFKA_CA_PATH: ../docker/kafka/ca.crt
  * KAFKA_PRIVATE_KEY_PATH: ../docker/kafka/client.key
  * KAFKA_PRIVATE_CERT_PATH: ../docker/kafka/client.crt
  * CLIENT_TOPIC: root_source.bush (if `NAME` in previous step was `bush`)
  * CLIENT_GROUP_ID: bush
  * CLIENT_ENCRYPTION_PRIVATE_KEY: a private part of pair which was generated at early step and is saved at `./id_rsa`
  * CLIENT_ENCRYPTION_PUBLIC_KEY: public key from iam. Get it by `curl -k -s -H "X-Vault-Token: TOKEN" https://127.0.0.1:8300/v1/flant/kafka/public_key | jq -r .data.public_key`
  * HTTP_URL: https://localhost:9200/asdf
```
- run rolebinding-watcher:
```
go run rolebinding-watcher/cmd/watcher/main.go
```
- run rolebinding-watcher e2e tests:
```
go test ./...
```
