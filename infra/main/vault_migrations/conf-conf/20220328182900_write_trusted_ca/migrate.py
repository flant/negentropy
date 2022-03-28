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

bucket_name = '%s-terraform-state' % google_credentials.project_id
vault_conf_ca_name = "vault-cert-auth-ca.pem"


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(vault_conf_ca_name)
    ca = str(blob.download_as_string(), 'utf-8')
    print("DEBUG: ca is", ca)
