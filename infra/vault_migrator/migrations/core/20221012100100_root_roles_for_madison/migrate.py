from typing import TypedDict, List, Type


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()

team_uuid_option_schema = '{"type":"object","required":["team_uuid"],"properties":{"team_uuid":{"pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"}}}'

# {role:{'scope':scope, 'tenant_is_optional':bool, 'project_is_optional':bool}}
roles = {
    'madison.team.manage': {'scope': 'tenant', 'tenant_is_optional': True, 'options_schema': team_uuid_option_schema},
    'madison.global_setups.manage': {'scope': 'tenant', 'tenant_is_optional': True},
    'madison.incidents.manage': {'scope': 'tenant', 'tenant_is_optional': True,
                                 'options_schema': team_uuid_option_schema},
    'madison.team_projects.manage': {'scope': 'tenant', 'tenant_is_optional': True,
                                     'options_schema': team_uuid_option_schema},
    'madison.incidents_notifications_settings.manage': {'scope': 'tenant', 'tenant_is_optional': False}
}


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)
    for role, kwargs in roles.items():
        print('INFO: create role `{}` at `{}` vault'.format(role, vault_name))
        kwargs['name'] = role
        vault.client.write(path='flant/role', **kwargs)
