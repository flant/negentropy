from typing import TypedDict, List

import hvac
import os


class Vault(TypedDict):
    name: str
    token: str
    url: str


kafka_endpoints_from_env = os.environ.get("NEGENTROPY_KAFKA_ENDPOINTS")
if kafka_endpoints_from_env is None:
    kafka_endpoints = "kafka:9092"
else:
    kafka_endpoints = kafka_endpoints_from_env

auth_vault_plugins = ['flant_iam_auth']
root_vault_plugins = ['flant_iam_auth', 'flant_iam']


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    if 'root' in vault_name:
        plugins = root_vault_plugins
    else:
        plugins = auth_vault_plugins
    for plugin in plugins:
        print("INFO: generate kafka csr for '{}' plugin at '{}' vault".format(plugin, vault_name))
        print("DEBUG: kafka_endpoints is", kafka_endpoints)
        if plugin == 'flant_iam_auth':
            vault_client.write(path='auth/flant_iam_auth/kafka/generate_csr')
            vault_client.write(path='auth/flant_iam_auth/kafka/configure_access', kafka_endpoints=kafka_endpoints)
        elif plugin == 'flant_iam':
            vault_client.write(path='flant_iam/kafka/generate_csr')
            vault_client.write(path='flant_iam/kafka/configure_access', kafka_endpoints=kafka_endpoints)
