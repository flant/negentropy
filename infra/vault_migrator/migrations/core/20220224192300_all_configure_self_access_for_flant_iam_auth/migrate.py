from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()

def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)
    print(f"INFO: create full policy at '{vault.url}' vault")
    vault.client.sys.create_or_update_policy(name="full",
                                             policy='path "*" {capabilities = ["create", "read", "update", "delete", "list"]}')
    print(f"INFO: enable auth method 'approle', getting secret_id and role_id at '{vault.name}' vault")
    enabled_auth_methods = vault.client.sys.list_auth_methods()
    if "approle/" not in enabled_auth_methods:
        vault.client.sys.enable_auth_method(method_type="approle")
    vault.client.auth.approle.create_or_update_approle(role_name="full", mount_point="approle",
                                                       secret_id_ttl="360h", token_ttl="15m",
                                                       token_policies=["full"])
    role_id = vault.client.auth.approle.read_role_id(role_name="full", mount_point="approle").get("data").get("role_id")
    secret_id = vault.client.auth.approle.generate_secret_id(role_name="full", mount_point="approle").get("data").get(
        "secret_id")
    print(f"INFO: configure auth/flant/configure_vault_access at '{vault.name}' vault")
    print(vault.client.write(path='auth/flant/configure_vault_access', vault_addr=vault.url,
                       vault_tls_server_name='vault_host',
                       role_name='full', secret_id_ttl='360h', approle_mount_point='/auth/approle/',
                       role_id=role_id, secret_id=secret_id, vault_cacert='')) # not_strictly_required_for_local_access
