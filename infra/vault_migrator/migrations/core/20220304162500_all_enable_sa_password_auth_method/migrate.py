from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)

    print("INFO: create auth method 'service_account_password' at '{}' vault".format(vault_name))
    vault.client.write(path='auth/flant/auth_method/sapassword', method_type='service_account_password',
                       token_ttl='30m', token_max_ttl='1440m',
                       user_claim='uuid', token_policies=["vst_owner", "list_tenants", "token_renew"],
                       token_no_default_policy='True')
