import json
import os
from google.auth import compute_engine
from google.oauth2 import service_account
from google.cloud import storage
from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


google_credentials_from_env = os.environ.get("GOOGLE_CREDENTIALS")
if google_credentials_from_env == None:
    google_credentials = compute_engine.Credentials()
else:
    google_credentials = service_account.Credentials.from_service_account_info(
        json.loads(os.environ.get("GOOGLE_CREDENTIALS")))

bucket_name = '%s-terraform-state' % google_credentials.project_id
vault_conf_ca_name = "vault-conf-ca.pem"


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)
    storage_client = storage.Client(credentials=google_credentials)
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(vault_conf_ca_name)
    ca = str(blob.download_as_string(), 'utf-8')
    print("INFO: upload vault conf trusted ca at '{}' vault".format(vault_name))
    vault.client.write(path='auth/cert/certs/auth', display_name='auth', policies='auth', certificate=ca, ttl='3600')
