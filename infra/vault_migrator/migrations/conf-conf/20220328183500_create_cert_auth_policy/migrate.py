from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)

    print("INFO: create cert auth policy at '{}' vault".format(vault_name))
    vault.client.sys.create_or_update_policy(name="cert-auth",
                                             policy="""path "*" {capabilities = ["create", "read", "update", "delete", "list", "sudo"]}""")
