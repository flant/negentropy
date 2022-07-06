# kafka-consumer

flant negentropy kafka-consumer daemon provide data transport from kafka into microservice endpoint

## Build

- git clone git@github.com:flant/negentropy.git
- cd negentropy/kafka-consumer
- GO111MODULE=on go build -o /OPT/kafka-consumer main.go

## Deployment

- /OPT/kafka-consumer should be delivered to server which have access both to negentropy kafka and microservice
- kafka-consumer should be configured:
  * kafka endpoint and topic
  * path to microservice endpoint

## Development

for tests and debug:

- run negentropy instance: from root project folder: ```./start.sh e2e```
- generate rsa key pair in PEM: run from kafka-consumer folder:

``` ssh-keygen  -t rsa -m PEM -N '' -f ./id_rsa  <<<y >/dev/null 2>&1 
ssh-keygen  -e -f ./id_rsa  -m PEM > ./id_rsa.pub ```
- get token for root-vault:   
  ```cat /tmp/vaults | jq '[.[]|select(.name=="root")][0].token'```
- create special replica kafka topic for consumer reading:
  ```

curl --location --request POST 'localhost:8300/v1/flant/replica/NAME' \
--header 'X-Vault-Token: TOKEN' \
--header 'Content-Type: application/json' \
--data-raw '{
"type":"Vault",
"public_key":"-----BEGIN RSA PUBLIC
KEY-----\\nMIIBigKCAYEA4cS4zynvKjYPzVVz921JXWLuElks/cs6CBvJK9UAWdapAg4P+Hb8\\ni2ZycG/r4UEjeffpfBQlwqbE75v29mpxhidE+c6Qs5zJfe5+lyIh0AW+m9TC9IFO\\n6o6NV/Z8foyH+oPzf1ZgKcuTXUc7xlRNK2niun9HJHzrUOLVN1CmBbwu0jyXY+Jq\\n8hl5NYsHLuvGwciyBLERtrIM6bp6a0fLl1ypsloZYW80MyTl7oX6V+sdoQlIIBcJ\\nlCevWMqn9NqhlFSCtL0fdQHJLXOqo6H6WZrEIwWbWGjd0iMTtXIcUPbZ04YUEtCf\\nlsV4YewaoXdANZDJRc798UeBuya8AjWiCt+4/TKdCjlpYmhJ2eCrAhGU0sAFoc81\\nmfJmJb/8OgfwOAzJ8BgGYshukwEXUvQX6V8P5EbTQT97N/rjPQyBFkZh61qv5+MM\\naiIfu2D/wOprDg2mibhehbMV7SarUdVLgIhd8FJ46CsA9riuAR0w0ICe5ndt2M6s\\n80Vn72rBbU47AgMBAAE=\\n-----END
RSA PUBLIC KEY-----"}'```
  * TOKEN is a token gotten at previous step (without quotes)
  * NAME is a name of replica, topic-name will be "root_source.NAME"
  * public_key is a public part of pair, which was generated at early step and save at ./id_rsa.pub
- pass following to consumer:
    * kafka-endpoint: localhost:9094
    * kafka-topic: root_source.NAME
    * ssl-config: // use_ssl, ca_path, client_private_key_path, client_certificate_path (files for access to
      docker-compose kafka are at ./docker/kafka)
    * private_key: is a public part of pair, which was generated at early step and is saved at ./id_rsa
    * public_key of iam/flant: get it
      by: ```curl -H "X-Vault-Token: TOKEN"  http://127.0.0.1:8300/v1/flant/kafka/public_key | jq .data.public_key```
      where TOKEN is a token gotten at previous step (without quotes)
    * topic: it is a name of topic which is created and encoded according to call /flant/replica/NAME : root_source.NAME
    * groupID: it is a kafka consumer reading group identifier
    * http gate url (just run ```go run kafka-consumer/e2e/mock/gate_mock.go``` and pass http://localhost:9200/asdf)
  