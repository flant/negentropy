from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


auth_vault_plugins = {'ssh': 'secret', 'flant_iam_auth': 'auth'}
root_vault_plugins = {'ssh': 'secret', 'flant_iam_auth': 'auth', 'flant_iam': 'secret'}


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: activate plugins at '{}' vault ".format(vault_name))
    active_auths = set(vault_client.sys.list_auth_methods().keys())
    active_secrets = set(vault_client.sys.list_mounted_secrets_engines().keys())
    plugins_to_activate = auth_vault_plugins
    if 'root' in vault_name:
        plugins_to_activate = root_vault_plugins
    for plugin_name, plugin_type in plugins_to_activate.items():
        if plugin_type == 'secret':
            if plugin_name + '/' not in active_secrets:
                vault_client.sys.enable_secrets_engine(
                    backend_type=plugin_name,
                    path=plugin_name,
                    plugin_name=plugin_name,
                )
                print("INFO: secret plugin '{}' is activated at '{}' vault".format(plugin_name, vault_name))
            else:
                print("INFO: secret plugin '{}' already activated at '{}' vault".format(plugin_name, vault_name))
        if plugin_type == 'auth':
            if plugin_name + '/' not in active_auths:
                vault_client.sys.enable_auth_method(
                    method_type=plugin_name,
                    path=plugin_name,
                    plugin_name=plugin_name,
                )
                print("INFO: auth plugin '{}' is activated at '{}' vault".format(plugin_name, vault_name))
            else:
                print("INFO: auth plugin '{}' already activated at '{}' vault".format(plugin_name, vault_name))
