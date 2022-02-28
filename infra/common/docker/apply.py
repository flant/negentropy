#!/usr/bin/env python3

# Daemon to use a Hardware TRNG
# apt install rng-tools

# GOOGLE_CREDENTIALS and GOOGLE_PROJECT environment variables must be set

import argparse
import base64
import json
import os
import subprocess
from typing import List, Dict

import gnupg
import hvac
import sys
import time
from google.auth import compute_engine
from google.cloud import storage
from google.oauth2 import service_account

from migrator import upgrade_vaults

google_credentials_from_env = os.environ.get("GOOGLE_CREDENTIALS")
if google_credentials_from_env == None:
    google_credentials = compute_engine.Credentials()
else:
    google_credentials = service_account.Credentials.from_service_account_info(
        json.loads(os.environ.get("GOOGLE_CREDENTIALS")))

google_project_id = google_credentials.project_id
terraform_state_bucket = '%s-terraform-state' % google_project_id
gnupghome = '/tmp/gnupg'

vault_auth_ca_name = "vault-cert-auth-ca.pem"

if not os.path.exists(gnupghome):
    os.makedirs(gnupghome)


def build_and_deploy(args: argparse.Namespace) -> Dict:
    """
    Build and run virtual machines and returns their ips
    :param args:
    :return: dict, keys are names of virtual_machines, vaules are static ip adresses
    example: {'private_static_ip_negentropy-kafka-1': 'X.Y.Z.31',
                'private_static_ip_negentropy-kafka-2': ...,
                'private_static_ip_negentropy-vault-auth-ew3a1': 'X.Y.Z.4',
                'private_static_ip_negentropy-vault-conf-conf': '10.20.3.2',
                'private_static_ip_negentropy-vault-root-source-1', ...}
    """
    print("• [base] run packer")
    run_bash("./build.sh", "../../base/packer")

    if args.type == 'configurator':
        print("• [configurator] run packer")
        run_bash("./build.sh", "../../configurator/packer")
        print("• [configurator] terraform apply")
        terraform_log = run_bash(
            "terraform init -backend-config bucket=%s-terraform-state; terraform apply -no-color -auto-approve" % google_project_id,
            "../../configurator/terraform")
    else:
        print("• [main] run packer")
        run_bash("./build.sh", "../../main/packer")
        print("• [main] terraform apply")
        terraform_log = run_bash(
            "terraform init -backend-config bucket=%s-terraform-state; terraform apply -no-color -auto-approve" % google_project_id,
            "../../main/terraform")
    write_file("/tmp/terraform_log", terraform_log)
    ips_map = {}
    for s in terraform_log.splitlines():
        if "private_static_ip_negentropy" in s:
            tmp = s.replace(' ', '').replace('"', '').split('=')
            ips_map[tmp[0]] = tmp[1]
    return ips_map


def get_vault_list_with_statuses(args: argparse.Namespace) -> (List, List):
    """
    Returns list of vaults names and, list of dicts, each dict contains keys: 'name':str and 'initialized':bool
    :param args:
    :return: [{'name':'auth-ew3a1', 'initialized':false}...]
    """
    vault_list = []
    if args.type == 'configurator':
        vault_list = next(os.walk('../../configurator/vault_migrations'))[1]
    elif args.type == 'main':
        vault_list = ['root-source-2', 'root-source-3', 'conf-conf', 'root-source-1', 'auth-ew3a1']
    else:
        print("--type %s not allow. Allow types: [configurator, main]" % args.type)
        sys.exit(1)
    vault_list_with_status = []
    for vault_name in vault_list:
        vault_list_with_status.append({'name': vault_name, 'initialized': check_blob_exists(terraform_state_bucket,
                                                                                            'negentropy-vault-' + vault_name + '-recovery-keys')})
    return vault_list, vault_list_with_status


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--type', dest='type', help='configurator or main')
    parser.add_argument('--target-migration-version', dest='target_migration_version')
    parser.add_argument('--save-root-tokens-on-initialization', dest='save_root_tokens', action='store_true')
    args = parser.parse_args()

    vault_list, vault_list_with_status = get_vault_list_with_statuses(
        args)  # [{'name':'auth-ew3a1', 'initialized':false}...]
    print("VAULT_LIST:", vault_list_with_status)

    ips_map = build_and_deploy(args)  # {'private_static_ip_negentropy-kafka-1': 'X.Y.Z.31',...}
    print("ips_map:", ips_map)

    for vault in vault_list_with_status:
        if not vault['initialized']:
            pgp_gen_key_and_upload_public_part('negentropy-vault-' + vault['name'] + '-temporary')

    for vault in vault_list_with_status:
        if not vault['initialized']:
            encrypted_vault_root_token_name = 'negentropy-vault-' + vault['name'] + '-root-token'
            while not check_blob_exists(terraform_state_bucket, encrypted_vault_root_token_name):
                print('root token for vault ' + vault['name'] + ' not found in bucket, sleep 2s')
                time.sleep(2)
            if args.save_root_tokens:
                encrypted_vault_root_token = download_blob_as_string(terraform_state_bucket,
                                                                     encrypted_vault_root_token_name)
                vault_root_token = str(pgp_decrypt(base64.b64decode(encrypted_vault_root_token)), 'utf-8')
                write_file(encrypted_vault_root_token_name + '-decrypted', vault_root_token)
        else:
            decrypted_vault_root_token_file = 'negentropy-vault-' + vault['name'] + '-root-token-decrypted'
            decrypted_vault_root_token = read_file(decrypted_vault_root_token_file)
            vault_root_token = decrypted_vault_root_token

        print("VAULT %s" % vault['name'])
        print("export VAULT_TOKEN=%s" % vault_root_token)
        if vault['name'] == 'auth-ew3a1':
            vault_address = 'https://%s:8200' % ips_map['private_static_ip_negentropy-vault-%s' % vault['name']]
        else:
            vault_address = 'https://%s:443' % ips_map['private_static_ip_negentropy-vault-%s' % vault['name']]
        print("export VAULT_ADDR=%s" % vault_address)
        print("export VAULT_CACERT=/usr/local/share/ca-certificates/negentropy-flant-local.pem")

    vaults_with_url_and_token = []
    if args.type == 'configurator':
        for vault in vault_list_with_status:
            decrypted_vault_root_token_name = 'negentropy-vault-' + vault['name'] + '-root-token-decrypted'
            decrypted_vault_root_token = read_file(decrypted_vault_root_token_name)
            vault_root_token = decrypted_vault_root_token
            vault_address = 'https://%s:443' % ips_map['private_static_ip_negentropy-vault-%s' % vault['name']]
            # os.environ['VAULT_TOKEN'] = vault_root_token
            # os.environ['VAULT_ADDR'] = vault_address
            print('DEBUG: vault_root_token is', vault_root_token)
            print('DEBUG: vault_address is', vault_address)

            while check_vault_is_ready(vault_url=vault_address, vault_token=vault_root_token) != True:
                time.sleep(1)

            # client = hvac.Client(url=vault_address, token=vault_root_token)
            # list_mounted_secrets_engines = client.sys.list_mounted_secrets_engines().keys()
            # if not 'secret/' in list_mounted_secrets_engines:
            #     client.sys.enable_secrets_engine(
            #         backend_type='kv',
            #         path='secret',
            #         options={'version': 1},
            #     )
            #
            # upgrade_db_command(type('obj', (object,),
            #                         {'migration_dir': '../../configurator/vault_migrations/%s' % vault['name'],
            #                          'version': args.target_migration_version}))
            vaults_with_url_and_token.append(
                {'name': vault['name'], 'url': vault_address, 'token': vault_root_token})
    else:
        vault_root_source_list = []
        for vault in vault_list:
            if vault.startswith("root-source"):
                vault_root_source_list.append(vault)
        print('DEBUG: vault_root_source_list is', vault_root_source_list)

        for vault in vault_root_source_list:
            decrypted_vault_root_source_token_file = 'negentropy-vault-' + vault + '-root-token-decrypted'
            if not check_file_empty(decrypted_vault_root_source_token_file):
                decrypted_vault_root_source_token = read_file(decrypted_vault_root_source_token_file)

        vault_root_source_standby_list = []
        for vault in vault_root_source_list:
            vault_address = 'https://%s:443' % ips_map['private_static_ip_negentropy-vault-%s' % vault]
            client = hvac.Client(url=vault_address)
            status = client.sys.read_leader_status()
            if not status['is_self']:
                vault_root_source_standby_list.append(vault)
        print('DEBUG: vault_root_source_standby_list is', vault_root_source_standby_list)

        vault_list_for_migrations = [vault for vault in vault_list_with_status if
                                     vault['name'] not in vault_root_source_standby_list]
        print('DEBUG: vault_list_for_migrations is', vault_list_for_migrations)

        for vault in vault_list_for_migrations:
            if vault['name'].startswith("root-source"):
                vault_root_token = decrypted_vault_root_source_token
            else:
                decrypted_vault_root_token_file = 'negentropy-vault-' + vault['name'] + '-root-token-decrypted'
                decrypted_vault_root_token = read_file(decrypted_vault_root_token_file)
                vault_root_token = decrypted_vault_root_token
            if vault['name'] == 'auth-ew3a1':
                vault_address = 'https://%s:8200' % ips_map['private_static_ip_negentropy-vault-%s' % vault['name']]
            else:
                vault_address = 'https://%s:443' % ips_map['private_static_ip_negentropy-vault-%s' % vault['name']]
            # os.environ['VAULT_TOKEN'] = vault_root_token
            # os.environ['VAULT_ADDR'] = vault_address
            print('DEBUG: vault_root_token is', vault_root_token)
            print('DEBUG: vault_address is', vault_address)

            while check_vault_is_ready(vault_url=vault_address, vault_token=vault_root_token) != True:
                time.sleep(1)

            # client = hvac.Client(timeout=5, url=vault_address, token=vault_root_token)
            # list_mounted_secrets_engines = client.sys.list_mounted_secrets_engines().keys()
            # if not 'secret/' in list_mounted_secrets_engines:
            #     client.sys.enable_secrets_engine(
            #         backend_type='kv',
            #         path='secret',
            #         options={'version': 1},
            #     )
            #
            # upgrade_db_command(type('obj', (object,),
            #                         {'migration_dir': '../../main/vault_migrations/%s' % vault['name'],
            #                          'version': args.target_migration_version}))

            vaults_with_url_and_token.append(
                {'name': vault['name'], 'url': vault_address, 'token': vault_root_token})
    print('vaults_with_url_and_token:', vaults_with_url_and_token)

    # migrator calling
    if args.type == 'configurator':
        migration_dir = '../../configurator/vault_migrations'
    elif args.type == 'main':
        migration_dir = '../../main/vault_migrations'
    upgrade_vaults(vaults_with_url_and_token, migration_dir)

def check_vault_is_ready(vault_url: str, vault_token: str):
    client = hvac.Client(timeout=5, url=vault_url, token=vault_token)
    try:
        health_status = client.sys.read_health_status()
        if health_status.status_code == 200:
            return (True)
    except Exception as e:
        print(e)
        return (False)
    return (False)


def check_blob_exists(bucket_name, blob_name):
    storage_client = storage.Client(credentials=google_credentials)
    blobs = storage_client.list_blobs(bucket_name)
    for blob in blobs:
        if blob.name == blob_name:
            return (True)
    return (False)


def check_bucket_exists(bucket_name):
    storage_client = storage.Client(credentials=google_credentials)
    buckets = storage_client.list_buckets()
    for bucket in buckets:
        if bucket.name == bucket_name:
            return (True)
    return (False)


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
    return (stdout)


def upload_blob_from_string(bucket_name, source_string, destination_blob_name):
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(destination_blob_name)
    blob.upload_from_string(source_string)
    print('file uploaded to gs://' + bucket_name + '/' + destination_blob_name)


def download_blob_as_string(bucket_name, blob_name):
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(blob_name)
    return (str(blob.download_as_string(), 'utf-8'))


def pgp_gen_key_and_upload_public_part(name):
    pgp_gen_key(name)
    pgp_public_key = pgp_get_public_key(name + '@flant.com')
    upload_blob_from_string(terraform_state_bucket, pgp_public_key, name + "-pub-key.asc")


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
                                   name_email=name + '@' + email_domain,
                                   no_protection=True)
    if not pgp_check_key_exists_by_name(name):
        gpg.gen_key(input_data)


def pgp_check_key_exists_by_name(name):
    gpg = gnupg.GPG(gnupghome=gnupghome)
    for key in gpg.list_keys():
        for uid in key['uids']:
            if name in uid:
                print('pgp key with name {} already exists'.format(name))
                return (True)
    return (False)


def pgp_get_public_key(key_id):
    gpg = gnupg.GPG(gnupghome=gnupghome)
    return (gpg.export_keys(key_id, expect_passphrase=False))
    # return(gpg.export_keys(key_id, secret=True, expect_passphrase=False)) # get secret key


def pgp_decrypt(input):
    gpg = gnupg.GPG(gnupghome=gnupghome)
    output = gpg.decrypt(input)
    return (output.data)


def write_file(path, data):
    f = open(path, 'w')
    f.write(data)
    f.close()


def read_file(path):
    f = open(path, 'r')
    return (f.read())


def check_file_empty(path):
    if os.path.getsize(path) == 0:
        return (True)
    else:
        return (False)


if __name__ == '__main__':
    main()
