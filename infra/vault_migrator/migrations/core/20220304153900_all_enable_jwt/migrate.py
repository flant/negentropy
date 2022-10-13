from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


auth_vault_plugins = ['flant_iam_auth']
root_vault_plugins = ['flant_iam_auth', 'flant_iam']


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)

    if 'root' in vault.name:
        plugins = root_vault_plugins
    else:
        plugins = auth_vault_plugins
    for plugin in plugins:
        print("INFO: enable jwt for '{}' plugin at '{}' vault".format(plugin,vault_name))
        if plugin == 'flant_iam_auth':
            vault.client.write(path='auth/flant/jwt/enable')
        elif plugin == 'flant_iam':
            vault.client.write(path='flant/jwt/enable')
