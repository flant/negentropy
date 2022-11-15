from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()

# {role:{'scope':scope, 'tenant_is_optional':bool, 'project_is_optional':bool}}
roles = {'billing.cfo': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.billing_methods.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.documents.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.invoices.generate': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.acts.generate': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.acts.delete': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.special_billings.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.sku.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.tariffs.manage': {'scope': 'tenant', 'tenant_is_optional': True},
         'billing.teams.manage': {'scope': 'tenant', 'tenant_is_optional': True}
         }


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)

    for role, kwargs in roles.items():
        print('INFO: create role `{}` at `{}` vault'.format(role, vault_name))
        kwargs['name'] = role
        vault.client.write(path='flant/role', **kwargs)
    cfo_include = ['billing.documents.manage',
                   'billing.invoices.generate',
                   'billing.acts.generate',
                   'billing.acts.delete',
                   'billing.special_billings.manage',
                   'billing.sku.manage',
                   'billing.tariffs.manage',
                   'billing.teams.manage']
    for role in cfo_include:
        vault.client.write(path=f'flant/role/billing.cfo/include/{role}')
