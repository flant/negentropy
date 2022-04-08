# Kafka with TLS

## Generate CA
```shell
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -days 365 -subj "/CN=negentropy"
```

## Create truststore from CA
```shell
keytool -keystore kafka.truststore.jks -alias CARoot -import -file ca.crt -storepass foobar -keypass foobar -noprompt
```

## Create keystore
```shell
keytool -keystore kafka.keystore.jks -alias localhost -validity 365 -genkey -keyalg RSA -dname "CN=kafka" -ext SAN=DNS:kafka,DNS:localhost,IP:127.0.0.1 -storepass foobar -keypass foobar -noprompt
```

## Generate CSR to the keystore
```shell
keytool -keystore kafka.keystore.jks -alias localhost -certreq -file server.csr -storepass foobar -keypass foobar
```

## Sign CSR
```shell
openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extensions req_ext -extfile config.cnf
```

## Import CA to keystore
```shell
keytool -keystore kafka.keystore.jks -alias CARoot -import -file ca.crt -storepass foobar -keypass foobar -noprompt
```

## Import signed certificate to keystore
```shell
keytool -keystore kafka.keystore.jks -alias localhost -import -file server.crt -noprompt -keypass foobar -storepass foobar
```

## Generate CSR for client
```shell
openssl req -nodes -newkey rsa:2048 -keyout client.key -out client.csr -subj "/CN=client"
```

## Sign client CSR
```shell
openssl x509 -req -CA ca.crt -CAkey ca.key -in client.csr -out client.crt -days 365 -CAcreateserial
```

---
For correct work SAN in signed certificates needs openssl config like this:
```
[ req ]
distinguished_name = req_distinguished_name
req_extensions     = req_ext

[ req_distinguished_name ]
commonName = Common Name (e.g. server FQDN or YOUR name)

[ req_ext ]
subjectAltName = @alt_names

[alt_names]
DNS.1 = kafka
DNS.2 = localhost
IP.1  = 127.0.0.1
```
