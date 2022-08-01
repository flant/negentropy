from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


# {role:{'scope':scope, 'tenant_is_optional':bool, 'project_is_optional':bool}}
roles = {'billing.cfo': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.billing-methods.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.documents.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.invoices.generate': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.acts.generate': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.acts.delete': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.special-billings.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.sku.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.tariffs.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.teams.manage': {'scope': 'tenant', 'tenant_is_optional': True}
         }


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    for role, kwargs in roles.items():
        print('INFO: create role `{}` at `{}` vault'.format(role, vault_name))
        kwargs['name'] = role
        vault_client.write(path='flant/role', **kwargs)
    cfo_include = ['billing.documents.manage',
                   'billing.invoices.generate',
                   'billing.acts.generate',
                   'billing.acts.delete',
                   'billing.special-billings.manage',
                   'billing.sku.manage',
                   'billing.tariffs.manage',
                   'billing.teams.manage']
    for role in cfo_include:
        vault_client.write(path=f'flant/role/billing.cfo/include/{role}')
