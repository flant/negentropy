from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


# {role:{'scope':scope, 'tenant_is_optional':bool, 'project_is_optional':bool}}
roles = {'pult.access.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'pult.team.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'pult.team_projects.manage': {'scope': 'tenant', 'tenant_is_optional': True}
         }


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    for role, kwargs in roles.items():
        print('INFO: create role `{}` at `{}` vault'.format(role, vault_name))
        kwargs['name'] = role
        vault_client.write(path='flant/role', **kwargs)
