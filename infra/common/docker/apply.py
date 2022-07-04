#!/usr/bin/env python3

import os
import json
import argparse
import gnupg
import sys
import yaml
import subprocess
import time
import base64
import hvac
import requests

from google.oauth2 import service_account
from google.cloud import storage

from migrator import upgrade_vaults


google_credentials = service_account.Credentials.from_service_account_info(json.loads(os.environ.get("GOOGLE_CREDENTIALS")))
google_project_id = google_credentials.project_id

terraform_state_bucket = '%s-terraform-state' % google_project_id

gnupghome = '/tmp/gnupg'
if not os.path.exists(gnupghome):
    os.makedirs(gnupghome)


def pgp_gen_key_and_upload_public_part(name: str):
    pgp_gen_key(name)
    pgp_public_key = pgp_get_public_key(name + '@flant.com')
    upload_blob_from_string(terraform_state_bucket, pgp_public_key, name + "-pub-key.asc")

def pgp_gen_key(name: str, email_domain: str = 'flant.com'):
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

def pgp_check_key_exists_by_name(name: str) -> bool:
    gpg = gnupg.GPG(gnupghome=gnupghome)
    for key in gpg.list_keys():
        for uid in key['uids']:
            if name in uid:
                print('pgp key with name {} already exists'.format(name))
                return (True)
    return (False)

def pgp_get_public_key(key_id: str) -> str:
    gpg = gnupg.GPG(gnupghome=gnupghome)
    return (gpg.export_keys(key_id, expect_passphrase=False))

def pgp_decrypt(input: bytes) -> bytes:
    gpg = gnupg.GPG(gnupghome=gnupghome)
    output = gpg.decrypt(input)
    return (output.data)

def upload_blob_from_string(bucket_name: str, source_string: str, destination_blob_name: str):
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(destination_blob_name)
    blob.upload_from_string(source_string)
    print('file uploaded to gs://' + bucket_name + '/' + destination_blob_name)

def download_blob_as_string(bucket_name: str, blob_name: str) -> str:
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(blob_name)
    return (str(blob.download_as_string(), 'utf-8'))

def check_blob_exists(bucket_name: str, blob_name: str) -> bool:
    storage_client = storage.Client(credentials=google_credentials)
    blobs = storage_client.list_blobs(bucket_name)
    for blob in blobs:
        if blob.name == blob_name:
            return (True)
    return (False)

def read_file(path: str) -> str:
    f = open(path, 'r')
    return (f.read())

def write_file(path: str, data: str):
    f = open(path, 'w')
    f.write(data)
    f.close()

def check_file_empty(path: str) -> bool:
    if os.path.getsize(path) == 0:
        return (True)
    else:
        return (False)

# TODO: fix wrong lines output order and add print output as optional parameter
def run_command(command: str, path: str = '.') -> str:
    sp = subprocess.Popen(command, cwd=path, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
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

def generate_packer_config(env_name: str):
    packer_config_path = '../config/environments/' + env_name + '.yaml'
    packer_variables_pkrvars_path = '../../variables.pkrvars.hcl'
    packer_config = list(yaml.safe_load_all(open(packer_config_path, "r")))
    packer_variables_pkrvars = open(packer_variables_pkrvars_path, 'w')
    print("generate packer variables.pkrvars.hcl using '{}' config file".format(packer_config_path))
    cfg = next(i for i in packer_config if i["kind"] == "PackerConfiguration")
    for k, v in cfg['config'].items():
        packer_variables_pkrvars.write('{}'" = "'"{}"'"\n".format(k,v))
    packer_variables_pkrvars.close

def generate_terraform_order(type: str, order: str) -> list:
    list = sorted(next(os.walk('../../'+type+'/terraform'))[1])
    if order == 'bootstrap':
        result = list
    elif order == 'update':
        if '99-load-balancers' in list:
            list.remove('99-load-balancers')
            for i in list:
                if ('vault-root-source' in i) or ('vault-auth' in i):
                    list.insert(list.index(i)+1, '99-load-balancers')
        result = list
    return (result)

def check_vault_is_ready(vault_url: str, attempts: int = 5) -> bool:
    for i in range(attempts):
        try:
            client = hvac.Client(url=vault_url)
            status = client.sys.is_initialized()
            return (status)
        except Exception:
            time.sleep(1)
            continue
    raise Exception("%s attempts were failed, vault at %s is unreachable" % (attempts, vault_url))

def check_kafka_is_ready(kafka_url: str, attempts: int = 10) -> bool:
    kafka_healthcheck_metric_name = 'kafka_server_kafkaserver_brokerstate'
    kafka_healthcheck_metric_value = '3.0'
    for i in range(attempts):
        try:
            response = requests.get(kafka_url).text
            for s in response.split("\n"):
                if s.startswith(kafka_healthcheck_metric_name):
                    if s.split(" ")[1].strip() == kafka_healthcheck_metric_value:
                        return (True)
        except Exception:
            time.sleep(1)
            continue
    raise Exception("%s attempts were failed, kafka at %s is unreachable or not ready" % (attempts, kafka_url))

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--type', dest='type', help='configurator or main', choices=['configurator', 'main'], required=True)
    parser.add_argument('--env', dest='env', required=True)
    parser.add_argument('--save-root-tokens-on-initialization', dest='save_root_tokens', action='store_true')
    parser.add_argument('--bootstrap', dest='bootstrap', action='store_true')
    args = parser.parse_args()

    if args.type == 'configurator':
        vault_list = ['conf']
    elif args.type == 'main':
        vault_list = ['conf-conf', 'root-source', 'auth-ew3a1']

    vault_list_with_status = []
    for vault_name in vault_list:
        vault_list_with_status.append({'name': vault_name, 'initialized': check_blob_exists(terraform_state_bucket, 'negentropy-vault-' + vault_name + '-recovery-keys')})
    print("DEBUG: vault_list_with_status is", vault_list_with_status)

    for vault in vault_list_with_status:
        if not vault['initialized']:
            pgp_gen_key_and_upload_public_part('negentropy-vault-' + vault['name'] + '-temporary')

    generate_packer_config(args.env)

    if args.bootstrap == True:
        terraform_order = generate_terraform_order(args.type, 'bootstrap')
    else:
        terraform_order = generate_terraform_order(args.type, 'update')
    print("DEBUG: terraform_order is", terraform_order)

    # TODO: add run terraform apply for base infrastructure
    print("• [base] run packer")
    run_command("./build.sh", "../../base/packer")

    print("• [configurator] run packer")
    run_command("./build.sh", "../../" + args.type + "/packer")

    terraform_config_path = '../config/environments/' + args.env + '.yaml'
    terraform_config = list(yaml.safe_load_all(open(terraform_config_path, "r")))
    cfg = next(i for i in terraform_config if i["kind"] == "TerraformConfiguration")
    for k, v in cfg['variables'].items():
        os.environ["TF_VAR_" + k] = str(v).lower()

    terraform_vault_output = {}
    for target in terraform_order:
        print("• [" + args.type + "] [%s] terraform init" % target)
        run_command("terraform init -backend-config bucket=%s-terraform-state" % google_project_id, "../../" + args.type + "/terraform/" + target)
        print("• [" + args.type + "] [%s] terraform plan" % target)
        run_command("terraform plan", "../../" + args.type + "/terraform/" + target)
        print("• [" + args.type + "] [%s] terraform apply" % target)
        run_command("terraform apply -auto-approve", "../../" + args.type + "/terraform/"+target)
        if 'vault' in target:
            vault_name_with_ip = json.loads(run_command("terraform output -json private_static_ip", "../../" + args.type + "/terraform/" + target))
            terraform_vault_output.update(vault_name_with_ip)
            if terraform_order == 'update':
                for k,v in vault_name_with_ip.items():
                    while check_vault_is_ready(vault_url='https://' + v + ':8200') != True:
                        time.sleep(1)
        if 'kafka' in target and terraform_order == 'update':
            kafka_name_with_ip = json.loads(run_command("terraform output -json private_static_ip", "../../" + args.type + "/terraform/" + target))
            for k,v in kafka_name_with_ip.items():
                while check_kafka_is_ready(kafka_url=v) != True:
                    time.sleep(1)
    print("DEBUG: terraform_vault_output is", terraform_vault_output)

    vault_urls = []
    for vault in terraform_vault_output:
        if 'root-source' not in vault:
            vault_urls.append({'name': vault, 'url': 'https://%s:8200' % terraform_vault_output[vault]})
        else:
            vault_url = 'https://%s:8200' % terraform_vault_output[vault]
            while check_vault_is_ready(vault_url=vault_url) != True:
                time.sleep(1)
            client = hvac.Client(url=vault_url)
            status = client.sys.read_leader_status()
            if status['is_self']:
                vault_root_source_master_url = vault_url
                print("DEBUG: vault_root_source_master_url is", vault_root_source_master_url)
                vault_urls.append({'name': 'root-source', 'url': vault_root_source_master_url})
    print("DEBUG: vault_urls is", vault_urls)

    vault_tokens = []
    for vault in vault_list_with_status:
        if not vault['initialized']:
            encrypted_vault_root_token_name = 'negentropy-vault-' + vault['name'] + '-root-token'
            while not check_blob_exists(terraform_state_bucket, encrypted_vault_root_token_name):
                print('root token for vault ' + vault['name'] + ' not found in bucket, sleep 2s')
                time.sleep(2)
            encrypted_vault_root_token = download_blob_as_string(terraform_state_bucket, encrypted_vault_root_token_name)
            vault_root_token = str(pgp_decrypt(base64.b64decode(encrypted_vault_root_token)), 'utf-8')
            vault_tokens.append({'name': vault['name'], 'token': vault_root_token})
            if args.save_root_tokens:
                write_file(encrypted_vault_root_token_name + '-decrypted', vault_root_token)
        else:
            # TODO: add check if the token file exists
            decrypted_vault_root_token_file = 'negentropy-vault-' + vault['name'] + '-root-token-decrypted'
            decrypted_vault_root_token = read_file(decrypted_vault_root_token_file)
            vault_root_token = decrypted_vault_root_token
            vault_tokens.append({'name': vault['name'], 'token': vault_root_token})
    print("DEBUG: vault_root_tokens is", vault_tokens)

    vault_list_with_urls_and_tokens = []
    for vault in vault_list:
        vault_name = vault
        vault_url = next(v['url'] for v in vault_urls if v['name'] == vault)
        vault_token = next(v['token'] for v in vault_tokens if v['name'] == vault)
        while check_vault_is_ready(vault_url=vault_url) != True:
            time.sleep(1)
        vault_list_with_urls_and_tokens.append({'name': vault_name, 'url': vault_url, 'token': vault_token})
    print("DEBUG: vault_list_with_urls_and_tokens is", vault_list_with_urls_and_tokens)

    migration_dir = None
    migration_config_path = '../config/environments/' + args.env + '.yaml'
    if args.type == 'configurator':
        migration_dir = '../../configurator/vault_migrations'
    elif args.type == 'main':
        migration_dir = '../../main/vault_migrations'
    upgrade_vaults(vault_list_with_urls_and_tokens, migration_dir, migration_config_path)


if __name__ == '__main__':
    main()
