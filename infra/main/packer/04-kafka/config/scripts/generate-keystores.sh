#!/usr/bin/env bash

. /etc/kafka/scripts/variables.sh

KEYSTORE_PATH="/data/keystore"
if  [[ -f ${KEYSTORE_PATH}/kafka.server.keystore.jks ]] && [[ ${KEYSTORE_PATH}/kafka.server.truststore.jks ]]; then
    echo "Keystores are present, update isn't need"
    exit 0
fi

mkdir -p ${KEYSTORE_PATH}

# Get CA root certificate
gcloud beta privateca roots describe ${KAFKA_GCP_CA_NAME} --location ${KAFKA_GCP_CA_LOCATION} --project ${GCP_PROJECT} --format='get(pemCaCertificates)' > ${KEYSTORE_PATH}/ca-cert

# Generating Private Key
keytool -genkey -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks -validity ${CERT_VALIDITY_DAYS} -storepass ${SERVER_KEY_PASS} -keypass ${SERVER_KEY_PASS} -dname "CN=${FQDN}" -storetype pkcs12 -keyalg RSA -ext SAN=DNS:${FQDN}

# Generating CSR
keytool -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks -certreq -file ${KEYSTORE_PATH}/csr -storepass ${SERVER_KEY_PASS} -keypass ${SERVER_KEY_PASS}

# TODO: maybe revoke old certs for this hostname, or download existing one.

# Getting CSR Signed with the CA
gcloud beta privateca certificates create ${HOSTNAME}-${TIMESTAMP} --issuer=${KAFKA_GCP_CA_NAME} --issuer-location=${KAFKA_GCP_CA_LOCATION} --csr ${KEYSTORE_PATH}/csr --cert-output-file ${KEYSTORE_PATH}/csr-signed --validity P${CERT_VALIDITY_DAYS}D --project ${GCP_PROJECT}

# Import CA certificate in KeyStore
keytool -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks -alias CARoot -import -file ${KEYSTORE_PATH}/ca-cert -storepass ${SERVER_KEY_PASS} -keypass ${SERVER_KEY_PASS} -noprompt

# Import Signed CSR In KeyStore
keytool -keystore ${KEYSTORE_PATH}/kafka.server.keystore.jks -import -file ${KEYSTORE_PATH}/csr-signed -storepass ${SERVER_KEY_PASS} -keypass ${SERVER_KEY_PASS} -noprompt

# Import CA certificate In TrustStore
keytool -keystore ${KEYSTORE_PATH}/kafka.server.truststore.jks -alias CARoot -import -file ${KEYSTORE_PATH}/ca-cert -storepass ${SERVER_KEY_PASS} -keypass ${SERVER_KEY_PASS} -noprompt

# Remove temporary files
rm -f ${KEYSTORE_PATH}/csr ${KEYSTORE_PATH}/csr-signed ${KEYSTORE_PATH}/csr ${KEYSTORE_PATH}/ca-cert
