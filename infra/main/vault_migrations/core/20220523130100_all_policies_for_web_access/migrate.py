from typing import TypedDict, List

import hvac
import time


class Vault(TypedDict):
    name: str
    token: str
    url: str

    #  [{'name': name,  'roles': [role1, role2,...], 'claim_schema': TODO,
    #   'allowed_auth_methods': ['method1', method2], 'rego_file':filepath
    #   }]


policies = [
    {'name': 'flant.teammate', 'roles': ['flant.teammate'], 'claim_schema': 'TODO', 'allowed_auth_methods': ['oidc'],
     'rego_file': 'flant.teammate.rego'},
    {'name': 'flant.admin', 'roles': ['flant.admin'], 'claim_schema': 'TODO', 'allowed_auth_methods': ['oidc'],
     'rego_file': 'flant.admin.rego'},
    {'name': 'flant.client.manage', 'roles': ['flant.client.manage'], 'claim_schema': 'TODO',
     'allowed_auth_methods': ['oidc'],
     'rego_file': 'flant.client.manage.rego'},
    {'name': 'tenant.manage', 'roles': ['tenant.manage'], 'claim_schema': 'TODO', 'allowed_auth_methods': ['oidc'],
     'rego_file': 'tenant.manage.rego'},
    {'name': 'tenant.read', 'roles': ['tenant.read'], 'claim_schema': 'TODO', 'allowed_auth_methods': ['oidc'],
     'rego_file': 'tenant.read.rego'}
]

# need in case of first run to provide time for roles appears at auth plugins
time.sleep(5)


def upgrade(vault_name: str, vaults: List[Vault]):
    import os
    folder = os.path.dirname(os.path.realpath(__file__))
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    for policy in policies:
        with open(os.path.join(folder, policy['rego_file']), "r") as f:
            policy['rego'] = f.read()
            print("INFO: create policy '{}' at '{}' vault".format(policy['name'], vault_name))
            vault_client.write(path='auth/flant/login_policy', **policy)
