import json
import sys
from os import path, makedirs
from typing import List

from flant_iam import create_role_if_not_exists, create_privileged_tenant, create_privileged_user, create_user_multipass, create_auth_method
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


if __name__ == "__main__":
    print(sys.argv)
    if "DEV" in sys.argv:
        dev_vault = Vault(name="vault_dev", url="http://127.0.0.1:8200", token="root",
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

    oidc_url = "https://login.flant.com"

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
        vault.activate_plugins()
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

    # ============================
    # export tokens
    # ============================
    write_tokens_files(vaults)

    # ================================
    # configuration for id_token login
    # ================================
    for vault in vaults:
        print("==========================================================")
        print("id_token login preparation: vault: {} at {}".format(vault.name, vault.url))
        print("----------------------------------------------------------")
        vault.connect_oidc(oidc_url)

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



    create_privileged_user(iam_vault, "00000991-0000-4000-A000-000000000000",
                           "e58cd98a-9421-4ead-8d7e-fc8abb4713a5",
                           "user_for_authd_tests_vad")
    multipass = create_user_multipass(iam_vault, "00000991-0000-4000-A000-000000000000",
                                      "e58cd98a-9421-4ead-8d7e-fc8abb4713a5", 3600)
    file = open(multipass_file_path, "w")
    file.write(multipass)
    file.close()
    print(multipass_file_path + " is updated")

    create_auth_method(iam_vault, "okta-jwt2")
