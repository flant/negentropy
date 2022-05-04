from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


flant_tenant_uuid = "be0ba0d8-7be7-49c8-8609-c62ac1f14597"
flant_identifier = "flant"
devops_team = "DevOps"
l1_team_name = "L1"
l1_team_id = "L1"
l1_team_uuid = "885909a2-a578-421f-b090-34273fdcadda"
mk8s_team_name = "mk8s"
mk8s_team_id = "foxtrot"
mk8s_team_uuid = "5b834d95-d2d2-4689-ab0e-dd31df49a748"
okmeter_team_name = "Okmeter"
okmeter_team_id = "okmeter"
okmeter_team_uuid = "3b896fc4-bf71-4a1d-8cf9-b8c665d6889f"
all_flant_group_uuid = "0d8dff50-474f-40f9-b431-d10e8e2c7dfc"


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: configure flant_flow extension at '{}' vault".format(vault_name))

    cfg = vault_client.read(path='flant_iam/configure_extension/flant_flow').get('data').get('flant_flow_cfg')

    print("INFO: creating tenant '{}' with uuid '{}'".format(flant_identifier, flant_tenant_uuid))
    flant = cfg.get('flant_tenant_uuid')
    if not flant or flant == '':
        vault_client.write(path='flant_iam/client/privileged', uuid=flant_tenant_uuid, identifier=flant_identifier)
        vault_client.write(path='flant_iam/configure_extension/flant_flow/flant_tenant/' + flant_tenant_uuid)

    print("INFO: creating group 'all@flant' with uuid '{}'".format(all_flant_group_uuid))
    all_flant_group = cfg.get('all_flant_group_uuid')
    if not all_flant_group or all_flant_group == '':
        vault_client.write(path='flant_iam/configure_extension/flant_flow/all_flant_group/' + all_flant_group_uuid)

    print("INFO: creating role rules")
    rules = cfg.get('roles_for_specific_teams')
    if not rules or not rules.get(devops_team):
        vault_client.write(path='flant_iam/configure_extension/flant_flow/role_rules/' + devops_team,
                           specific_roles=['ssh.open'])

    print("INFO: configure teams")
    teams = cfg.get('specific_teams')
    if not teams.get(l1_team_name):
        vault_client.write(path='flant_iam/team/privileged', uuid=l1_team_uuid, identifier=l1_team_id,
                           team_type='standard_team')
        vault_client.write(path='flant_iam/configure_extension/flant_flow/specific_teams',
                           specific_teams={l1_team_name: l1_team_uuid})
        print("INFO: team '{}' with uuid '{}' created".format(l1_team_name, l1_team_uuid))
    if not teams.get(mk8s_team_name):
        vault_client.write(path='flant_iam/team/privileged', uuid=mk8s_team_uuid, identifier=mk8s_team_id,
                           team_type='standard_team')
        vault_client.write(path='flant_iam/configure_extension/flant_flow/specific_teams',
                           specific_teams={mk8s_team_name: mk8s_team_uuid})
        print("INFO: team '{}' with uuid '{}' created".format(mk8s_team_name, mk8s_team_uuid))
    if not teams.get(okmeter_team_name):
        vault_client.write(path='flant_iam/team/privileged', uuid=okmeter_team_uuid, identifier=okmeter_team_id,
                           team_type='standard_team')
        vault_client.write(path='flant_iam/configure_extension/flant_flow/specific_teams',
                           specific_teams={okmeter_team_name: okmeter_team_uuid})
        print("INFO: team '{}' with uuid '{}' created".format(okmeter_team_name, okmeter_team_uuid))
