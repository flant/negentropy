
import os
from typing import List, Tuple

from vault import Vault
from config import log, flant_tenant_uuid, proto_team_uuid, PRIVATE_KEY, PUBLIC_KEY, multipass_file_path


def create_privileged_teammate_to_proto_team(vault: Vault, teammate_uuid: str, teammate_email: str):
    """create privileged user if not exists"""
    base_path = f"flant/team/{proto_team_uuid}/teammate/"
    role_binding_path = f"flant/tenant/{flant_tenant_uuid}/role_binding"
    
    resp = vault.client.read(path=base_path + teammate_uuid)
    
    if resp:
        log.info(f"teammate with uuid '{teammate_uuid}' already exists")
        return
    
    vault.client.write(path=base_path + "privileged", uuid=teammate_uuid, identifier=teammate_email.split("@")[0],
                       email=teammate_email, role_at_team="member")
    log.info(f"teammate with uuid '{teammate_uuid}' created")

    resp = vault.client.write(path=role_binding_path, description=f"flant.admin for primary user: {teammate_email}",
                              members=[{"uuid": teammate_uuid, "type": "user",}],
                              roles=[{"name": "flant.admin", "options": {},}])
    log.info(f"'flant.admin' role for teammate with uuid '{teammate_uuid}' is available")
    
    resp = vault.client.write(path= role_binding_path, description=f"tenant.manage for primary user: {teammate_email}",
                              members=[{"uuid": teammate_uuid, "type": "user",}],
                              roles=[{"name": "tenant.manage", "options": {}}])
    log.info(f"'tenant.manage' role for teammate with uuid '{teammate_uuid}' is available")


def create_privileged_team_proto(vault: Vault):
    """create proto team for first teammate"""
    
    resp = vault.client.read(path="flant/team/" + proto_team_uuid)
    if resp:
        log.info(f"proto team uuid '{proto_team_uuid}' already exists")
        return
    vault.client.write(path='flant/team/privileged', uuid=proto_team_uuid, identifier="proto",
                       team_type='standard_team')
    log.info(f"proto team uuid  '{proto_team_uuid}' created")
    

def write_tokens_to_file(vaults: List[Vault]):
    if len(vaults) > 1:
        for vault in vaults:
            f = open("/tmp/vault_" + vault.name + "_token", "w")
            f.write(vault.token)
            f.close()
    else:
        f = open("/tmp/vault_single_token", "w")
        f.write(vaults[0].token)
        f.close()


def split_vaults(vaults: List[Vault]) -> Tuple[List[Vault], List[Vault]]:
    """
    splits passed vaults into two groups
    :param vaults:
    :return: two lists of vaults: first list contains auth and root-source vaults, second one - others
    """
    core_vaults = []
    other_vaults = []

    for v in vaults:
        if any(sub_name in v.name for sub_name in ['auth', 'root']):
            core_vaults.append(v)
        else:
            other_vaults.append(v)
    return core_vaults, other_vaults


def prepare_user_multipass_jwt(vault:Vault):
    # ============================================================================
    # prepare user multipass_jwt for authd tests
    # ============================================================================   
    
    if not vault:
        raise Exception(f"there is no vaults with flant_iam onboard in passed!")
    
    vault.create_privileged_tenant("00000991-0000-4000-A000-000000000000", "tenant_for_authd_tests")   
    vault.create_privileged_user("00000991-0000-4000-A000-000000000000",
                           "00000661-0000-4000-A000-000000000000",
                           "user_for_authd_tests@gmail.com")
    multipass = vault.create_user_multipass("00000991-0000-4000-A000-000000000000",
                                      "00000661-0000-4000-A000-000000000000", 3600)
    
    
    multipass_file_folder = multipass_file_path.rsplit("/", 1)[0]
    
    if not os.path.exists(multipass_file_folder):
        os.makedirs(multipass_file_folder)
    file = open(multipass_file_path, "w")
    file.write(multipass)
    file.close()
    
    log.info(multipass_file_path + " is updated")
    

def create_teammate_for_webdev(vault: Vault,args):
    # ============================================================================
    # create teammate for webdev local development
    # ============================================================================
    
        log.debug("OKTA UUID is", args.okta_uuid)
        log.debug("OKTA EMAIL IS", args.okta_email)
        
        create_privileged_team_proto(vault)
        create_privileged_teammate_to_proto_team(vault, args.okta_uuid, args.okta_email)



def single_mode_code_run(vault:Vault):

    log.info(f"import ssh ca at '{vault.name}' vault")

    vault.client.write(path='ssh/config/ca', private_key=PRIVATE_KEY, public_key=PUBLIC_KEY)

    log.info(f"create ssh signer at '{vault.name}' vault")
    
    vault.client.write(path='ssh/roles/signer', allow_user_certificates=True, algorithm_signer='rsa-sha2-256',
                    allowed_users='*', allowed_extensions='permit-pty,permit-agent-forwarding', key_type='ca',
                    default_extensions=[{"permit-agent-forwarding": "", "permit-pty": ""}], ttl='5m0s')