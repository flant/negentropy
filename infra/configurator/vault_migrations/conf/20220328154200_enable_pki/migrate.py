from typing import TypedDict, List

import hvac
import json
import os
from google.auth import compute_engine
from google.oauth2 import service_account
from google.cloud import storage


class Vault(TypedDict):
    name: str
    token: str
    url: str


google_credentials_from_env = os.environ.get("GOOGLE_CREDENTIALS")
if google_credentials_from_env == None:
    google_credentials = compute_engine.Credentials()
else:
    google_credentials = service_account.Credentials.from_service_account_info(
        json.loads(os.environ.get("GOOGLE_CREDENTIALS")))

vault_conf_pki_name = "vault-cert-auth"
vault_conf_ca_name = "vault-conf-ca.pem"
bucket_name = '%s-terraform-state' % google_credentials.project_id


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: enable pki at '{}' vault".format(vault_name))
    enabled_secrets = vault_client.sys.list_mounted_secrets_engines()
    if vault_conf_pki_name + '/' not in enabled_secrets:
        vault_client.sys.enable_secrets_engine(backend_type='pki', path=vault_conf_pki_name, max_lease_ttl='87600h')
    vault_client.write(path=vault_conf_pki_name + '/roles/auth', allow_any_name='true', max_ttl='1h')
    print("INFO: generate and upload ca at '{}' vault".format(vault_name))
    ca_response = vault_client.secrets.pki.read_ca_certificate(mount_point=vault_conf_pki_name)
    if not ca_response:
        ca = vault_client.write(path=vault_conf_pki_name + '/root/generate/internal', common_name='negentropy', ttl='87600h').get('data').get('issuing_ca')
    else:
        print("INFO: ca already exists at '{}' vault".format(vault_name))
        ca_response = ca
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(vault_conf_ca_name)
    blob.upload_from_string(str(ca))
    print('INFO: file uploaded to gs://' + bucket_name + '/' + vault_conf_ca_name)
