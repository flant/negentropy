from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


# {role:{'scope':scope, 'tenant_is_optional':bool, 'project_is_optional':bool}}
roles = {'planner.all.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'carter.users.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'carter.task_categories.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         }


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)
    for role, kwargs in roles.items():
        print('INFO: create role `{}` at `{}` vault'.format(role, vault_name))
        kwargs['name'] = role
        vault.client.write(path='flant/role', **kwargs)
