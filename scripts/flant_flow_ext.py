import requests

from consts import FLANT_IAM, FLANT_TENANT_UUID, FLANT, L1_TEAM_UUID, L1_TEAM_ID, MK8S_TEAM_UUID, MK8S_TEAM_ID, \
    OKMETER_TEAM_UUID, OKMETER_TEAM_ID
from flant_iam import create_privileged_tenant
from vault import Vault, check_response

L1_TEAM = "L1"
MK8S_TEAM = "mk8s"
OKMETER_TEAM = "Okmeter"
DEVOPS_TEAM = "DevOps"

STANDARD_TEAM_TYPE = "standard_team"


class FlantFlowExtension:
    def __init__(
            self,
            root_vault: Vault,
    ):
        if root_vault is None:
            raise Exception("definitely needs root_vault")
        self.root_vault = root_vault
        """Creates a new FlantFlowExtension instance for configuring plugin flant_iam.
        :param root_vault: root_vault instance for configuring extension
        :type root_vault: Vault        
        """

    def get_flant_flow_cfg(self) -> requests.Response:
        resp = check_response(
            self.root_vault.read_from_plugin(plugin=FLANT_IAM, path="configure_extension/flant_flow"))
        data = resp.json().get('data')
        if not data:
            raise Exception(
                "expected 'data' key, got {}\n".format(resp.text))
        cfg = data.get('flant_flow_cfg')
        if not cfg:
            raise Exception(
                "expected 'flant_flow_cfg' key, got {}\n".format(data))
        return cfg

    def configure_extension_at_root_vault(self):
        print("Flant_flow: configure_extension_at_root_vault:")
        self.configure_flant()
        self.configure_role_rules()
        self.configure_teams()

    def configure_flant(self):
        print("\tconfigure_flant", end=" ")
        cfg = self.get_flant_flow_cfg()
        flant = cfg.get('flant_tenant_uuid')
        if not flant or flant == '':
            create_privileged_tenant(self.root_vault, FLANT_TENANT_UUID, FLANT)
            self.root_vault.write_to_plugin(FLANT_IAM,
                                            "configure_extension/flant_flow/flant_tenant/" + FLANT_TENANT_UUID)
        print('... ok')

    def configure_role_rules(self):
        print("\tconfigure_role_rules", end=" ")
        cfg = self.get_flant_flow_cfg()
        rules = cfg.get('roles_for_specific_teams')
        if not rules or not rules.get(DEVOPS_TEAM):
            self.root_vault.write_to_plugin(FLANT_IAM, "configure_extension/flant_flow/role_rules/" + DEVOPS_TEAM,
                                            json={
                                                'specific_roles': ['ssh']
                                            })
        print('... ok')

    def configure_teams(self):
        print("\tconfigure_teams", end=" ")
        cfg = self.get_flant_flow_cfg()
        teams = cfg.get('specific_teams')
        if not teams or not teams.get(L1_TEAM):
            create_privileged_team(self.root_vault, L1_TEAM_UUID, L1_TEAM_ID, STANDARD_TEAM_TYPE)
            self.root_vault.write_to_plugin(FLANT_IAM, "configure_extension/flant_flow/specific_teams", json={
                'specific_teams': {L1_TEAM: L1_TEAM_UUID}
            })
        if not teams.get(MK8S_TEAM):
            create_privileged_team(self.root_vault, MK8S_TEAM_UUID, MK8S_TEAM_ID, STANDARD_TEAM_TYPE)
            self.root_vault.write_to_plugin(FLANT_IAM, "configure_extension/flant_flow/specific_teams", json={
                'specific_teams': {MK8S_TEAM: MK8S_TEAM_UUID}
            })
        if not teams.get(OKMETER_TEAM):
            create_privileged_team(self.root_vault, OKMETER_TEAM_UUID, OKMETER_TEAM_ID, STANDARD_TEAM_TYPE)
            self.root_vault.write_to_plugin(FLANT_IAM, "configure_extension/flant_flow/specific_teams", json={
                'specific_teams': {OKMETER_TEAM: OKMETER_TEAM_UUID}
            })
        print('... ok')


def create_privileged_team(vault: Vault, team_uuid: str, team_identifier: str, team_type: str):
    """create team if not exists"""
    resp = vault.read_from_plugin(plugin=FLANT_IAM, path="team/" + team_uuid)
    if resp.status_code == 200:
        print("team with uuid '{}' already exists".format(team_uuid))
        return
    if resp.status_code == 404:
        check_response(
            vault.write_to_plugin(plugin=FLANT_IAM, path="team/privileged",
                                  json={"uuid": team_uuid, "identifier": team_identifier, "team_type": team_type}), 201)
        print("team with uuid '{}' created".format(team_uuid))
        return
    raise Exception("expect one of status :[200, 404], got: {}, full response:{}".format(resp.status_code, resp.text))
