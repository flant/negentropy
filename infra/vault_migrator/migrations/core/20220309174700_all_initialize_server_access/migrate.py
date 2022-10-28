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
        print("INFO: configure server_access extension for plugin '{}' at '{}' vault".format(plugin, vault_name))
        if plugin == 'flant_iam_auth':
            vault.client.write(path='auth/flant/configure_extension/server_access',
                               role_for_ssh_access='ssh.open')
        elif plugin == 'flant_iam':
            vault.client.write(path='flant/configure_extension/server_access', roles_for_servers=["server"],
                               role_for_ssh_access='ssh', delete_expired_password_seeds_after='1000000',
                               expire_password_seed_after_reveal_in='1000000', last_allocated_uid='10000')
