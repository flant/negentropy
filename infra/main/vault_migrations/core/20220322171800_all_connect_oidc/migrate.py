import os
from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


oidc_url = os.environ.get("NEGENTROPY_OIDC_URL")
if oidc_url == None:
    raise Exception("ERROR: NEGENTROPY_OIDC_URL must be set")


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: create auth source 'okta-oidc' at '{}' vault".format(vault_name))
    vault_client.write(path='auth/flant/auth_source/okta-oidc', oidc_discovery_url=oidc_url,
                       default_role='demo', entity_alias_name='full_identifier')
    print("INFO: create auth method 'okta-jwt' at '{}' vault".format(vault_name))
    vault_client.write(path='auth/flant/auth_method/okta-jwt', method_type='access_token', source='okta-oidc',
                       bound_audiences='https://login.flant.com', token_ttl='30m', token_max_ttl='1440m',
                       user_claim='email',
                       token_policies=["vst_owner", "list_tenants", "token_renew", "check_permissions"],
                       token_no_default_policy='True')
