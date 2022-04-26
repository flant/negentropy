from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: create policy for rotate multipass at '{}' vault".format(vault_name))
    vault_client.sys.create_or_update_policy(name="rotate_multipass",
                                             policy="""path "auth/flant_iam_auth/issue/multipass_jwt/*" {capabilities = ["update"]}"""
                                             )
    vault_client.sys.create_or_update_policy(name="read_auth",
                                             policy="""path "auth/flant_iam_auth/query_server" {capabilities = ["read"]} 
                                                   path "auth/flant_iam_auth/tenant/*" {capabilities = ["read","list"]}""")

    print("INFO: configure multipass at '{}' vault".format(vault_name))
    vault_client.write(path='auth/flant_iam_auth/auth_method/multipass', token_ttl='30m', token_max_ttl='1440m',
                       token_policies='vst_owner, list_tenants, token_renew, rotate_multipass, read_auth',
                       token_no_default_policy='True',
                       method_type='multipass_jwt')
