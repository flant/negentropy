from typing import TypedDict, List

import hvac
import os

class Vault(TypedDict):
    name: str
    token: str
    url: str


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: create full policy at '{}' vault".format(vault_name))
    vault_client.sys.create_or_update_policy(name="full",
                                             policy='path "*" {capabilities = ["create", "read", "update", "delete", "list"]}')
    print("INFO: enable auth method 'approle', getting secret_id and role_id at '{}' vault".format(vault_name))
    enabled_auth_methods = vault_client.sys.list_auth_methods()
    if "approle/" not in enabled_auth_methods:
        vault_client.sys.enable_auth_method(method_type="approle")
    vault_client.auth.approle.create_or_update_approle(role_name="full", mount_point="approle",
                                                       secret_id_ttl="15m", token_ttl="180s",
                                                       token_policies=["full"])
    role_id = vault_client.auth.approle.read_role_id(role_name="full", mount_point="approle").get("data").get("role_id")
    secret_id = vault_client.auth.approle.generate_secret_id(role_name="full", mount_point="approle").get("data").get("secret_id")
    print("INFO: configure auth/flant_iam_auth/configure_vault_access at '{}' vault".format(vault_name))
    vault_client.write(path='auth/flant_iam_auth/configure_vault_access', vault_addr=vault['url'],
                       vault_tls_server_name='vault_host',
                       role_name='full', secret_id_ttl='15m', approle_mount_point='/auth/approle/',
                       role_id=role_id, secret_id=secret_id, vault_api_ca='')
