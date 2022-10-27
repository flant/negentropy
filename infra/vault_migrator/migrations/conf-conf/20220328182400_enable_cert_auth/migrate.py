from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)
    print("INFO: enable 'cert' auth method at '{}' vault".format(vault_name))
    vault.client.sys.enable_auth_method(method_type='cert')
