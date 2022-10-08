# Purpose of tls folder content:

- provide key and cert (tls.key, tls.crt) for running vaults with tls (https) - they are passed by ../vault-xxxx.hcl
- provide ca.crt for making https requests which should be runned with verifying

## Content:

- ca.crt : root public CA
- tls.key, tls.crt: private key and public certificate for run HTTPS listener at:
    - localhost
    - vault
    - vault-root
    - vault-auth
    - vault-conf

## How to interact with vault https listener:

A) pass by VAULT_CACERT path to ca.crt (in case of using hashicorp clients)   
B) pass path or content of ca.crt to used library    
C) some-how turn-off verifying in tls config of client

A - is used for running migrations at ./start.sh C - is used in e2e tests clients

## How to recreate keys:

```openssl genrsa -des3 -out rootCA.key 4096 ``` => private key at file rootCA.key  
```openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out ca.crt``` => public CA certificate at file
ca.crt  
```openssl genrsa -out tls.key 2048``` =>  private key at file tls.key

```
openssl req -new -sha256 \
    -key tls.key \
    -subj "/CN=negentropy.vaults" \
    -reqexts SAN \
    -config <(cat /etc/ssl/openssl.cnf <(printf "\n[SAN]\nsubjectAltName=DNS:vault,DNS:vault-auth,DNS:vault-root,DNS:vault-conf,DNS:localhost,IP:127.0.0.1")) \
    -out tls.csr
```

=> certificate request at file tls.csr

```openssl x509 -req -extfile  <(pr.crtintf "subjectAltName=DNS:vault,DNS:vault-auth,DNS:vault-root,DNS:vault-conf,DNS:localhost,IP:127.0.0.1") -in tls.csr -CA ca.crt -CAkey rootCA.key -CAcreateserial -out tls.crt -days 1500 -sha256```

=> certificate signed by rootCA at tls.crt

__only ```ca.crt``` , ```tls.key```, ```tls.crt``` are used__