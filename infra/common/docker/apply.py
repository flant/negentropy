#!/usr/bin/env python3

# Daemon to use a Hardware TRNG
# apt install rng-tools

# GOOGLE_CREDENTIALS and GOOGLE_PROJECT environment variables must be set

import gnupg
import subprocess
import sys
import json
import os
import argparse
import time
import base64

from google.cloud import storage
from google.oauth2 import service_account
from google.auth import compute_engine

google_credentials_from_env = os.environ.get("GOOGLE_CREDENTIALS")
if google_credentials_from_env == None:
    google_credentials = compute_engine.Credentials()
else:
    google_credentials = service_account.Credentials.from_service_account_info(json.loads(os.environ.get("GOOGLE_CREDENTIALS")))

terraform_state_bucket = 'negentropy-terraform-state'
gnupghome = '/tmp/gnupg'

if not os.path.exists(gnupghome):
    os.makedirs(gnupghome)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--type', dest='type', help='configurator or main')
    parser.add_argument('--max-migration', dest='max_migration')
    parser.add_argument('--save-root-tokens-on-initialization', dest='save_root_tokens', action='store_true')
    args = parser.parse_args()

    if args.type == 'configurator':
        pgp_gen_key_and_upload_public_part('alice')
        pgp_gen_key_and_upload_public_part('bob')
        pgp_gen_key_and_upload_public_part('carol')

    vault_list = []
    if args.type == 'configurator':
        vault_list = next(os.walk('../../configurator/vault_migrations'))[1]
    elif args.type == 'main':
        vault_list = next(os.walk('../../main/vault_migrations'))[1]
    else:
      print("--type %s not allow. Allow types: [configurator, main]" % args.type)
      sys.exit(1)

    vault_list_with_status = []
    for vault_name in vault_list:
        vault_list_with_status.append({'name': vault_name, 'initialized': check_blob_exists(terraform_state_bucket, 'vault-'+vault_name+'-recovery-keys')})

    print(vault_list_with_status)

    os.environ['PKR_VAR_root_password'] = "d9eWkemNTe"

    os.environ['PKR_VAR_gcp_builder_service_account'] = "negentropy-packer@flant-sandbox.iam.gserviceaccount.com"
    os.environ['PKR_VAR_gcp_image_bucket'] = "negentropy-packer"

    os.environ['PKR_VAR_gcp_vault_root_source_bucket'] = "vault-root-source-1"
    os.environ['PKR_VAR_gcp_vault_conf_bucket'] = "vault-conf"
    os.environ['PKR_VAR_gcp_vault_conf_conf_bucket'] = "vault-conf-conf"
    os.environ['PKR_VAR_gcp_vault_auth_bucket_trailer'] = "vault-auth"

    os.environ['PKR_VAR_gcp_project'] = google_credentials.project_id
    os.environ['PKR_VAR_gcp_zone'] = "europe-west3-a"

    os.environ['PKR_VAR_kafka_main_domain'] = "c.flant-sandbox.internal"
    os.environ['PKR_VAR_kafka_server_key_pass'] = "Flant123"
    os.environ['PKR_VAR_kafka_bucket'] = "negentropy-kafka"
    os.environ['PKR_VAR_kafka_gcp_ca_name'] = "kafka-root-ca"
    os.environ['PKR_VAR_kafka_gcp_ca_location'] = "europe-west1"
    os.environ['PKR_VAR_kafka_replicas'] = "3"

    write_file("/tmp/credentials", os.environ.get('GOOGLE_CREDENTIALS'))
    os.environ['CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE'] = "/tmp/credentials"
    os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = "/tmp/credentials"

    pkrvars = f'''
# Should be the same during bas image build and others.
root_password = "{os.environ.get('PKR_VAR_root_password')}"
# Following two are used only for alpine-base-builder instance.
gcp_builder_service_account = "{os.environ.get('PKR_VAR_gcp_builder_service_account')}"
gcp_image_bucket = "{os.environ.get('PKR_VAR_gcp_image_bucket')}"
# FQDN buckets for single instance vaults.
gcp_vault_root_source_bucket = "{os.environ.get('PKR_VAR_gcp_vault_root_source_bucket')}"
gcp_vault_conf_bucket = "{os.environ.get('PKR_VAR_gcp_vault_conf_bucket')}"
gcp_vault_conf_conf_bucket = "{os.environ.get('PKR_VAR_gcp_vault_conf_conf_bucket')}"
# Will be used as "hostname.gcp_vault_auth_bucket_trailer".
gcp_vault_auth_bucket_trailer = "{os.environ.get('PKR_VAR_gcp_vault_auth_bucket_trailer')}"
# Variables to determine where are builder instances will run.
## Project will be also used for kafka CA request.
gcp_project = "{os.environ.get('PKR_VAR_gcp_project')}"
gcp_zone = "{os.environ.get('PKR_VAR_gcp_zone')}"
# Will be used as "hostname.kafka_main_domain".
kafka_main_domain = "{os.environ.get('PKR_VAR_kafka_main_domain')}"
# Kafka server key password.
kafka_server_key_pass = "{os.environ.get('PKR_VAR_kafka_server_key_pass')}"
# Bucket to store cert update lockfile.
kafka_bucket = "{os.environ.get('PKR_VAR_kafka_bucket')}"
# Name of root CA for kafka SSL.
kafka_gcp_ca_name = "{os.environ.get('PKR_VAR_kafka_gcp_ca_name')}"
# Root CA location (aka gcp region).
kafka_gcp_ca_location = "{os.environ.get('PKR_VAR_kafka_gcp_ca_location')}"
# How many replicas to configure in zookeeper and kafka configuration files.
kafka_replicas = "{os.environ.get('PKR_VAR_kafka_replicas')}"
###
gcp_region = "europe-west3"
gcp_ckms_seal_key_ring = "vault-vs-test"
gcp_ckms_seal_crypto_key = "vault-vs-test-crypto-key"
'''


    write_file("../../variables.pkrvars.hcl", pkrvars)

    print("• [common] run packer")
    run_bash("./build.sh", "../../common/packer")

    # print("• [common] build_vault")
    # run_bash("./build.sh", "../../common/vault/build_vault.sh")

    terraform_log = ''
    if args.type == 'configurator':
        print("• [configurator] run packer")
        run_bash("./build.sh", "../../configurator/packer")
        print("• [configurator] terraform apply")
        terraform_log = run_bash("terraform init; terraform apply -no-color -auto-approve", "../../configurator/terraform")
    else:
        print("• [main] run packer")
        run_bash("./build.sh", "../../main/packer")
        print("• [main] terraform apply")
        terraform_log = run_bash("terraform init; terraform apply -no-color -auto-approve", "../../main/terraform")

    write_file("/tmp/terraform_log", terraform_log)

    ips_map = {}
    for s in terraform_log.splitlines():
      if "private_static_ip_negentropy" in s:
        tmp = s.replace(' ', '').replace('"', '').split('=')
        ips_map[tmp[0]] = tmp[1]
    print(ips_map)

    for vault in vault_list_with_status:
        if not vault['initialized']:
            pgp_gen_key_and_upload_public_part('temporary-vault-'+vault['name'])

    for vault in vault_list_with_status:
        vault_root_token = ''
        if vault['name'] == 'conf':
            vault_conf_token = os.environ.get('VAULT_CONF_TOKEN')
            if vault_conf_token != None:
                vault_root_token = vault_conf_token

        if not vault['initialized']:
            encrypted_vault_root_token_name = 'vault-'+vault['name']+'-root-token'
            while not check_blob_exists(terraform_state_bucket, encrypted_vault_root_token_name):
                print('root token for vault '+vault['name']+' not found in bucket, sleep 2s')
                time.sleep(2)
            if args.save_root_tokens:
                encrypted_vault_root_token = download_blob_as_string(terraform_state_bucket, encrypted_vault_root_token_name)
                vault_root_token = str(pgp_decrypt(base64.b64decode(encrypted_vault_root_token)), 'utf-8')
                write_file(encrypted_vault_root_token_name+'-decrypted', vault_root_token)

        print("VAULT_TOKEN:", vault_root_token)

        vault_address = 'http://%s:8200' % ips_map['private_static_ip_negentropy-vault-%s' % vault['name']]
        print("VAULT_ADDR:", vault_address)

        if vault['name'] == 'conf':
            os.environ['VAULT_TOKEN'] = vault_root_token
            os.environ['VAULT_ADDR'] = vault_address
            run_bash("""
                     vault secrets enable pki;
                     vault secrets tune -max-lease-ttl=87600h pki;
                     """)






def check_blob_exists(bucket_name, blob_name):
    storage_client = storage.Client(credentials=google_credentials)
    blobs = storage_client.list_blobs(bucket_name)
    for blob in blobs:
        if blob.name == blob_name:
            return(True)
    return(False)

def check_bucket_exists(bucket_name):
    storage_client = storage.Client(credentials=google_credentials)
    buckets = storage_client.list_buckets()
    for bucket in buckets:
        if bucket.name == bucket_name:
            return(True)
    return(False)

def run_bash(script, path='.'):
    sp = subprocess.Popen(script, cwd=path, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout = str(sp.stdout.read(), 'utf-8')
    stderr = str(sp.stderr.read(), 'utf-8')
    if stdout != '':
        print(stdout)
    if stderr != '':
        print(stderr)
    exitcode = sp.wait()
    if exitcode != 0:
        sys.exit(exitcode)
    return(stdout)

def upload_blob_from_string(bucket_name, source_string, destination_blob_name):
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(destination_blob_name)
    blob.upload_from_string(source_string)
    print('file uploaded to gs://'+bucket_name+'/'+destination_blob_name)

def download_blob_as_string(bucket_name, blob_name):
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(blob_name)
    return(str(blob.download_as_string(), 'utf-8'))

def pgp_gen_key_and_upload_public_part(name):
        pgp_gen_key(name)
        pgp_public_key = pgp_get_public_key(name+'@flant.com')
        upload_blob_from_string(terraform_state_bucket, pgp_public_key, name+"-pub-key.asc")

# https://docs.red-dove.com/python-gnupg/
def pgp_gen_key(name, email_domain='flant.com'):
    gpg = gnupg.GPG(gnupghome=gnupghome)
    gpg.encoding = 'utf8'
    input_data = gpg.gen_key_input(key_type='RSA',
                                   key_length=2048,
                                   subkey_type='RSA',
                                   subkey_length=2048,
                                   name_real=name,
                                   name_comment=name,
                                   name_email=name+'@'+email_domain,
                                   no_protection=True)
    if not pgp_check_key_exists_by_name(name):
        gpg.gen_key(input_data)

def pgp_check_key_exists_by_name(name):
    gpg = gnupg.GPG(gnupghome=gnupghome)
    for key in gpg.list_keys():
        for uid in key['uids']:
            if name in uid:
                print('pgp key with name {} already exists'.format(name))
                return(True)
    return(False)

def pgp_get_public_key(key_id):
    gpg = gnupg.GPG(gnupghome=gnupghome)
    return(gpg.export_keys(key_id, expect_passphrase=False))
    # return(gpg.export_keys(key_id, secret=True, expect_passphrase=False)) # get secret key

def pgp_decrypt(input):
    gpg = gnupg.GPG(gnupghome=gnupghome)
    output = gpg.decrypt(input)
    return(output.data)

def write_file(path, data):
    f = open(path, 'w')
    f.write(data)
    f.close()

if __name__ == '__main__':
    main()
