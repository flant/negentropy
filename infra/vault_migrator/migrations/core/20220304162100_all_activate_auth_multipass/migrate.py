from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)

    print("INFO: create policy for rotate multipass at '{}' vault".format(vault_name))
    vault.client.sys.create_or_update_policy(name="rotate_multipass",
                                             policy="""path "auth/flant/issue/multipass_jwt/*" {capabilities = ["update"]}"""
                                             )
    vault.client.sys.create_or_update_policy(name="read_auth",
                                             policy="""path "auth/flant/query_server" {capabilities = ["read"]} 
                                                   path "auth/flant/tenant/*" {capabilities = ["read","list"]}""")

    print("INFO: configure multipass at '{}' vault".format(vault_name))
    vault.client.write(path='auth/flant/auth_method/multipass', token_ttl='30m', token_max_ttl='1440m',
                       token_policies='vst_owner, list_tenants, token_renew, rotate_multipass, read_auth',
                       token_no_default_policy='True',
                       method_type='multipass_jwt')
