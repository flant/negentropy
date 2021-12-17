from vault import Vault, check_response
from consts import FLANT_IAM


def create_role_if_not_exists(vault: Vault, role_name: str):
    """create role if not exists"""
    resp = vault.read_from_plugin(plugin=FLANT_IAM, path="role/" + role_name)
    if resp.status_code == 200:
        print("role '{}' already exists".format(role_name))
        return
    if resp.status_code == 404:
        check_response(
            vault.write_to_plugin(plugin=FLANT_IAM, path="role", json={"name": role_name, "scope": "tenant"}), 201)
        print("role '{}' created".format(role_name))
        return
    raise Exception("expect one of status :[200, 404], got: {}, full response:{}".format(resp.status_code, resp.text))
