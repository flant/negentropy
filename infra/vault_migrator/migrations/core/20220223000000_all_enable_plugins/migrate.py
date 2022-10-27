from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


# plugins = {plugin_name:{"type":plugin_type, "path":plugin_path}}

auth_vault_plugins = {'ssh': {'type': 'secret', 'path': 'ssh'}, 'flant_iam_auth': {'type': 'auth', 'path': 'flant'}}
root_vault_plugins = {'ssh': {'type': 'secret', 'path': 'ssh'}, 'flant_iam_auth': {'type': 'auth', 'path': 'flant'},
                      'flant_iam': {'type': 'secret', 'path': 'flant'}}


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)

    print("INFO: activate plugins at '{}' vault ".format(vault_name))
    active_auths = vault.client.sys.list_auth_methods()
    active_secrets = vault.client.sys.list_mounted_secrets_engines()
    if 'root' in vault.name:
        plugins_to_activate = root_vault_plugins
    else:
        plugins_to_activate = auth_vault_plugins
    for plugin_name, plugin in plugins_to_activate.items():
        plugin_type = plugin['type']
        plugin_path = plugin['path']
        if plugin_type == 'secret':
            if plugin_name + '/' not in active_secrets:
                vault.client.sys.enable_secrets_engine(
                    backend_type=plugin_name,
                    path=plugin_path,
                    plugin_name=plugin_path,
                )
                print("INFO: secret plugin '{}' is activated at '{}' vault".format(plugin_name, vault_name))
            else:
                print("INFO: secret plugin '{}' already activated at '{}' vault".format(plugin_name, vault_name))
        if plugin_type == 'auth':
            if plugin_name + '/' not in active_auths:
                vault.client.sys.enable_auth_method(
                    method_type=plugin_name,
                    path=plugin_path,
                    plugin_name=plugin_path,
                )
                print("INFO: auth plugin '{}' is activated at '{}' vault".format(plugin_name, vault_name))
            else:
                print("INFO: auth plugin '{}' already activated at '{}' vault".format(plugin_name, vault_name))
