from typing import TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str


def upgrade(vault_name: str, vaults: List[Vault]):
    print('===infra/main/vault_migrations/core/20220222222224_root_fake/migrate.py===')
    vault = next(v for v in vaults if v['name'] == vault_name)
    print(vault)
    # vault_client = hvac.Client(url=vault['url'], token=vault['token'])
