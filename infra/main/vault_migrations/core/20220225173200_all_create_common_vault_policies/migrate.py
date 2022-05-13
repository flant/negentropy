from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: create token renew policy at '{}' vault".format(vault_name))
    vault_client.sys.create_or_update_policy(name="token_renew",
                                             policy="""path "auth/token/lookup-self" {capabilities = ["create", "update", "read"]} path "auth/token/renew-self" {capabilities = ["create", "update", "read"]} """)
    print("INFO: create vst_owner policy at '{}' vault".format(vault_name))
    vault_client.sys.create_or_update_policy(name="vst_owner",
                                             policy="""path "auth/flant/vst_owner" {capabilities = ["read"]}""")
    print("INFO: create list_tenants policy at '{}' vault".format(vault_name))
    vault_client.sys.create_or_update_policy(name="list_tenants",
                                             policy="""path "auth/flant/tenant/" {capabilities = ["list"]}""")
