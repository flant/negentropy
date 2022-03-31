from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


roles = ['ssh', 'servers']


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    for role in roles:
        print("INFO: create role '{}' at '{}' vault".format(role, vault_name))
        vault_client.write(path='flant_iam/role', name=role, scope='tenant')
