from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


# {role:{'scope':scope, 'tenant_is_optional':bool, 'project_is_optional':bool}}
roles = {'ssh.open': {'scope': 'project', 'tenant_is_optional': True, 'project_is_optional': True,
                      'enriching_extensions': 'server_access'},
         'servers.query': {'scope': 'project', 'tenant_is_optional': True, 'project_is_optional': True},
         'tenant.read.auth': {'scope': 'tenant'},
         'servers.register': {'scope': 'project'},
         'server': {'scope': 'project'},
         # it is stub role, to use if nothing specific is needed
         'tenants.list.auth': {'scope': 'tenant', 'tenant_is_optional': True}}


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)

    for role, kwargs in roles.items():
        print('INFO: create role `{}` at `{}` vault'.format(role, vault_name))
        kwargs['name'] = role
        vault.client.write(path='flant/role', **kwargs)

    vault.client.write(path='flant/role/ssh.open/include/servers.query')
    vault.client.write(path='flant/role/ssh.open/include/tenant.read.auth')

    vault.client.write(path='flant/role/servers.register/include/tenant.read.auth')
