import os
from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


kafka_configure_options = {}

kafka_endpoints = os.environ.get("NEGENTROPY_KAFKA_ENDPOINTS")
if not kafka_endpoints:
    raise Exception("ERROR: NEGENTROPY_KAFKA_ENDPOINTS must be set")
kafka_configure_options['kafka_endpoints'] = kafka_endpoints

kafka_use_ssl = os.environ.get("NEGENTROPY_KAFKA_USE_SSL")
if not kafka_use_ssl:
    raise Exception("ERROR: NEGENTROPY_KAFKA_USE_SSL must be set")
kafka_configure_options['use_ssl'] = kafka_use_ssl

if kafka_use_ssl == "true":
    kafka_ssl_ca_path = os.environ.get("NEGENTROPY_KAFKA_SSL_CA_PATH")
    if not kafka_ssl_ca_path:
        raise Exception("ERROR: NEGENTROPY_KAFKA_SSL_CA_PATH must be set")
    kafka_configure_options['ca_path'] = kafka_ssl_ca_path

    kafka_ssl_client_private_key_path = os.environ.get("NEGENTROPY_KAFKA_SSL_CLIENT_PRIVATE_KEY_PATH")
    if not kafka_ssl_client_private_key_path:
        raise Exception("ERROR: NEGENTROPY_KAFKA_SSL_CLIENT_PRIVATE_KEY_PATH must be set")
    kafka_configure_options['client_private_key_path'] = kafka_ssl_client_private_key_path

    kafka_ssl_client_certificate_path = os.environ.get("NEGENTROPY_KAFKA_SSL_CLIENT_CERTIFICATE_PATH")
    if not kafka_ssl_client_certificate_path:
        raise Exception("ERROR: NEGENTROPY_KAFKA_SSL_CLIENT_CERTIFICATE_PATH must be set")
    kafka_configure_options['client_certificate_path'] = kafka_ssl_client_certificate_path

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
        print("INFO: configure kafka access for '{}' plugin at '{}' vault".format(plugin, vault_name))
        if plugin == 'flant_iam_auth':
            vault_client.write(path='auth/flant/kafka/configure_access', **kafka_configure_options)
        elif plugin == 'flant_iam':
            vault_client.write(path='flant/kafka/configure_access', **kafka_configure_options)
