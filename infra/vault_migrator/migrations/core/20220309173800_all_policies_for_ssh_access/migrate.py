import time
from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()

#  [{'name': name,  'roles': [role1, role2,...], 'claim_schema': TODO,
#   'allowed_auth_methods': ['method1', method2], 'rego_file':filepath
#   }]
policies = [
    {'name': 'ssh.open', 'roles': ['ssh.open'], 'claim_schema': 'TODO', 'allowed_auth_methods': ['multipass'],
     'rego_file': 'ssh.open.rego'},
    {'name': 'servers.query', 'roles': ['servers.query'], 'claim_schema': 'TODO',
     'allowed_auth_methods': ['multipass', 'sapassword'],
     'rego_file': 'servers.query.rego'},
    {'name': 'tenant.read.auth', 'roles': ['tenant.read.auth'], 'claim_schema': 'TODO',
     'allowed_auth_methods': ['multipass', 'sapassword'],
     'rego_file': 'tenant.read.auth.rego'},
    {'name': 'tenants.list.auth', 'roles': ['tenants.list.auth'], 'claim_schema': 'TODO',
     'allowed_auth_methods': ['multipass', 'sapassword', 'okta-jwt'],
     'rego_file': 'tenants.list.auth.rego'},
    {'name': 'servers.register', 'roles': ['servers.register'], 'claim_schema': 'TODO',
     'allowed_auth_methods': ['sapassword'],
     'rego_file': 'servers.register.rego'},
    {'name': 'server', 'roles': ['server'], 'claim_schema': 'TODO',
     'allowed_auth_methods': ['multipass'],
     'rego_file': 'server.rego'},

]

# need in case of first run to provide time for roles appears at auth plugins
time.sleep(5)


def upgrade(vault_name: str, vaults: List[Vault]):
    time.sleep(5)  # provide time to collect roles through kafka topic
    import os
    folder = os.path.dirname(os.path.realpath(__file__))
    vault = next(v for v in vaults if v.name == vault_name)
    
    for policy in policies:
        with open(os.path.join(folder, policy['rego_file']), "r") as f:
            policy['rego'] = f.read()
            print("INFO: create policy '{}' at '{}' vault".format(policy['name'], vault_name))
            vault.client.write(path='auth/flant/login_policy', **policy)
