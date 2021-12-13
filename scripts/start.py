from plugins import connect_plugins
from vault import Vault

root_vault = Vault(name="root", url="http://127.0.0.1:8300", token="root",
                   plugin_names=['flant_iam', 'flant_iam_auth', 'ssh'])
auth_vault = Vault(name="auth", url="http://127.0.0.1:8200", token="root",
                   plugin_names=['flant_iam_auth', 'ssh'])

vaults = [root_vault, auth_vault]


# Initialize vaults and plugins
for vault in vaults:
    vault.wait()
    vault.init_and_unseal()
    vault.activate_plugins()

connect_plugins(vaults, "kafka:9092")


def write_tokens_files(vaults):
    for vault in vaults:
        f = open("/tmp/" + vault.name + "_token", "w")
        f.write(vault.token)
        f.close()


write_tokens_files(vaults)
