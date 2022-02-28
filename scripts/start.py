import argparse
import importlib as importlib
import json
from os import path, makedirs
from typing import List

from flant_flow_ext import FlantFlowExtension
from flant_iam import create_role_if_not_exists, create_privileged_tenant, create_privileged_user, create_user_multipass
from plugins import connect_plugins
from plugins import find_master_root_vault
from server_access_ext import ServerAccessExtension
from vault import Vault


def initialize_server_access(vaults: List[Vault]):
    """ Initialize extension server-access """
    master_vault = find_master_root_vault(vaults)
    create_role_if_not_exists(master_vault, "ssh")
    create_role_if_not_exists(master_vault, "servers")
    server_access_extension = ServerAccessExtension(vaults=vaults, roles_for_servers=["servers"],
                                                    role_for_ssh_access="ssh")
    server_access_extension.configure_extension_at_vaults()


def initialize_flant_flow(vaults: List[Vault]):
    """ Initialize extension flant_flow """
    master_vault = find_master_root_vault(vaults)
    flant_flow_extension = FlantFlowExtension(root_vault=master_vault)
    flant_flow_extension.configure_extension_at_root_vault()


def write_tokens_files(vaults: List[Vault]):
    for vault in vaults:
        f = open("/tmp/" + vault.name + "_token", "w")
        f.write(vault.token)
        f.close()


def write_vaults_state_to_file(vaults: List[Vault]):
    file = open("/tmp/vaults", "w")
    out = []
    for v in vaults:
        out.append(v.marshall())
    file.write(json.dumps(out))
    file.close


def read_vaults_from_file() -> List[Vault]:
    file = open("/tmp/vaults", "r")
    s = file.read()
    file.close()
    cfgs = json.loads(s)
    vaults = []
    for cfg in cfgs:
        vaults.append(Vault(
            name=cfg['name'],
            url=cfg["url"],
            token=cfg['token'],
            plugin_names=cfg['plugin_names'],
            keys=cfg['keys']
        ))
    return vaults


def run_migrations(vaults: List[Vault]):
    # module_path = './infra/main/vault_migrations/core/run_all_migrations.py'
    module_path = './infra/common/docker/migrator.py'
    module_name = 'migrations'
    loader = importlib.machinery.SourceFileLoader(module_name, module_path)
    module = loader.load_module()
    # migration_dir = 'infra/main/vault_migrations/core'
    migration_dir = 'infra/main/vault_migrations'
    # module.run_all_migrations([{'name': v.name, 'url': v.url, 'token': v.token} for v in vaults], migration_dir)
    module.upgrade_vaults([{'name': v.name, 'url': v.url, 'token': v.token} for v in vaults], migration_dir)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', dest='mode')
    parser.add_argument('--oidc-url', dest='oidc_url')
    parser.add_argument('--okta-uuid', dest='okta_uuid')
    args = parser.parse_args()

    if args.mode == 'dev':
        dev_vault = Vault(name="root", url="http://127.0.0.1:8200", token="root",
                          plugin_names=['flant_iam', 'flant_iam_auth', 'ssh'])
        vaults = [dev_vault]
        auth_vault_name = dev_vault.name
    else:
        root_vault = Vault(name="root", url="http://127.0.0.1:8300", token="root",
                           plugin_names=['flant_iam', 'flant_iam_auth', 'ssh'])
        auth_vault = Vault(name="auth", url="http://127.0.0.1:8200", token="root",
                           plugin_names=['flant_iam_auth', 'ssh'])
        vaults = [root_vault, auth_vault]
        auth_vault_name = auth_vault.name

    oidc_url = args.oidc_url

    print("DEBUG: OIDC URL is", oidc_url)

    # vaults = read_vaults_from_file()

    # ============================
    # Initialize vaults and plugins
    # ============================
    for vault in vaults:
        print("========================================")
        print("vault: {} at {}".format(vault.name, vault.url))
        print("----------------------------------------")
        vault.wait()
        vault.init_and_unseal()

    run_migrations(vaults)

    for vault in vaults:
        # vault.activate_plugins()
        vault.configure_self_access_for_flant_iam_auth()
        vault.create_token_renew_policy()

    write_vaults_state_to_file(vaults)

    plugins = connect_plugins(vaults, "kafka:9092")

    # ============================
    # configuration for ssh-access
    # ============================
    for vault in vaults:
        print("==========================================================")
        print("ssh-access preparation: vault: {} at {}".format(vault.name, vault.url))
        print("----------------------------------------------------------")
        vault.activate_plugins_jwt()  # need kafka
        vault.activate_auth_multipass()  # need activate jwt
        vault.configure_ssh_ca([auth_vault_name])  # for using at ssh access tests

    initialize_server_access(vaults)
    initialize_flant_flow(vaults)
    # ============================
    # export tokens
    # ============================
    write_tokens_files(vaults)

    # ================================
    # configuration for id_token login
    # ================================
    for vault in vaults:
        print("==============================================================================")
        print("id_token and service_account_password login preparation: vault: {} at {}".format(vault.name, vault.url))
        print("------------------------------------------------------------------------------")
        vault.connect_oidc(oidc_url)
        vault.activate_auth_service_account_pass()

    # ============================================================================
    # logs (only requests done by vault.write_to_plugin or vault.read_from_plugin)
    # ============================================================================
    f = open("/tmp/vaults_logs.txt", "w")
    for v in vaults:
        f.write("requests logs for vault: {} at: {}\n".format(v.name, v.url))
        f.write("=====================\n")
        for line in v.get_request_log():
            f.write(str(line) + "\n")
        f.write("\n\n")
    f.close()

    # ============================================================================
    # prepare user multipass_jwt for authd tests
    # ============================================================================
    multipass_file_path = "authd/dev/secret/authd.jwt"
    multipass_file_folder = multipass_file_path.rsplit("/", 1)[0]
    if not path.exists(multipass_file_folder):
        makedirs(multipass_file_folder)
    iam_vault = find_master_root_vault(vaults)
    create_privileged_tenant(iam_vault, "00000991-0000-4000-A000-000000000000", "tenant_for_authd_tests")
    create_privileged_user(iam_vault, "00000991-0000-4000-A000-000000000000",
                           "00000661-0000-4000-A000-000000000000",
                           "user_for_authd_tests")
    multipass = create_user_multipass(iam_vault, "00000991-0000-4000-A000-000000000000",
                                      "00000661-0000-4000-A000-000000000000", 3600)
    file = open(multipass_file_path, "w")
    file.write(multipass)
    file.close()
    print(multipass_file_path + " is updated")

    # ============================================================================
    # create privileged user for webdev local development
    # ============================================================================

    if args.okta_uuid:
        print("DEBUG: OKTA UUID is", args.okta_uuid)
        create_privileged_user(iam_vault, "b2c3d385-6bc7-43ff-9e75-441330442b1e",
                               args.okta_uuid,
                               "local-admin")
        create_user_multipass(iam_vault, "b2c3d385-6bc7-43ff-9e75-441330442b1e",
                              args.okta_uuid, 3600)
