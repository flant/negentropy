from server_access_ext import ServerAccessExtension
from plugins import connect_plugins
from vault import Vault
from plugins import find_master_root_vault
from flant_iam import create_role_if_not_exists


def initialize_server_access(vaults: list[Vault]):
    """ Initialize extension server-access """
    master_vault = find_master_root_vault(vaults)
    create_role_if_not_exists(master_vault, "ssh")
    create_role_if_not_exists(master_vault, "servers")
    server_access_extension = ServerAccessExtension(vaults=vaults, roles_for_servers=["servers"],
                                                    role_for_ssh_access="ssh")
    server_access_extension.configure_extension_at_vaults()


def write_tokens_files(vaults):
    for vault in vaults:
        f = open("/tmp/" + vault.name + "_token", "w")
        f.write(vault.token)
        f.close()


root_vault = Vault(name="root", url="http://127.0.0.1:8300", token="root",
                   plugin_names=['flant_iam', 'flant_iam_auth', 'ssh'])
auth_vault = Vault(name="auth", url="http://127.0.0.1:8200", token="root",
                   plugin_names=['flant_iam_auth', 'ssh'])

vaults = [root_vault, auth_vault]

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

plugins = connect_plugins(vaults, "kafka:9092")

# ============================
# configuration for ssh-access
# ============================
for vault in vaults:
    print("========================================")
    print("vault: {} at {}".format(vault.name, vault.url))
    print("----------------------------------------")
    vault.activate_plugins_jwt()  # need kafka
    vault.activate_auth_multipass()  # need activate jwt
    vault.configure_ssh_ca([auth_vault.name])  # for using at

initialize_server_access(vaults)

# ============================
# export tokens
# ============================
write_tokens_files(vaults)

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
