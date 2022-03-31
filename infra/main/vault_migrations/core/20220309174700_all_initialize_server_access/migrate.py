from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


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
        print("INFO: configure server_access extension for plugin '{}' at '{}' vault".format(plugin, vault_name))
        if plugin == 'flant_iam_auth':
            vault_client.write(path='auth/flant_iam_auth/configure_extension/server_access', role_for_ssh_access='ssh')
        elif plugin == 'flant_iam':
            vault_client.write(path='flant_iam/configure_extension/server_access', roles_for_servers=["servers"],
                               role_for_ssh_access='ssh', delete_expired_password_seeds_after='1000000',
                               expire_password_seed_after_reveal_in='1000000', last_allocated_uid='10000')
