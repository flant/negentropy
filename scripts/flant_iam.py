from consts import FLANT_IAM
from vault import Vault, check_response


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


def create_privileged_tenant(vault: Vault, tenant_uuid: str, identifier: str):
    """create tenant if not exists"""
    resp = vault.read_from_plugin(plugin=FLANT_IAM, path="tenant/" + tenant_uuid)
    if resp.status_code == 200:
        print("tenant with uuid '{}' already exists".format(tenant_uuid))
        return
    if resp.status_code == 404:
        check_response(
            vault.write_to_plugin(plugin=FLANT_IAM, path="tenant/privileged",
                                  json={"uuid": tenant_uuid, "identifier": identifier}), 201)
        print("tenant with uuid '{}' created".format(tenant_uuid))
        return
    raise Exception("expect one of status :[200, 404], got: {}, full response:{}".format(resp.status_code, resp.text))


def create_privileged_user(vault: Vault, tenant_uuid: str, user_uuid: str, identifier: str):
    """create user if not exists"""
    base_path = "tenant/{}/user/".format(tenant_uuid)
    resp = vault.read_from_plugin(plugin=FLANT_IAM, path=base_path + user_uuid)
    if resp.status_code == 200:
        print("user with uuid '{}' already exists".format(user_uuid))
        return
    if resp.status_code == 404:
        check_response(
            vault.write_to_plugin(plugin=FLANT_IAM, path=base_path + "privileged",
                                  json={"uuid": user_uuid, "identifier": identifier}), 201)
        print("user with uuid '{}' created".format(user_uuid))
        return
    raise Exception("expect one of status :[200, 404], got: {}, full response:{}".format(resp.status_code, resp.text))


def create_user_multipass(vault: Vault, tenant_uuid: str, user_uuid: str, ttl_sec: int) -> str:
    """create user multipass"""
    resp = check_response(
        vault.write_to_plugin(plugin=FLANT_IAM, path="tenant/{}/user/{}/multipass".format(tenant_uuid, user_uuid),
                              json={"ttl": ttl_sec}), 201)
    body = resp.json()
    if type(body) is not dict:
        raise Exception("expect dict, got:{}".format(type(body)))
    if not body['data'] or \
            type(body['data']) is not dict \
            or not body['data']['token']:
        raise Exception("expect {'data':{'token':'xxxx', ...}, ...} dict, got:\n" + body)
    return body['data']['token']


def add_user_to_group(vault: Vault, tenant_uuid: str, user_uuid: str, group_uuid: str):
    """add user to group"""
    path = "tenant/{}/group/{}".format(tenant_uuid, group_uuid)
    resp = vault.read_from_plugin(plugin=FLANT_IAM, path=path)
    if resp.status_code != 200:
        raise Exception("got status={} expect 200, body:\n{}".format(resp.status_code, resp.text))
    body = resp.json()
    group = body['data']['group']
    print(group['users'])
    print(group['members'])
    if user_uuid not in group['users']:
        group['members'].append({'type': 'user', 'uuid': user_uuid})
        resp = vault.write_to_plugin(plugin=FLANT_IAM, path=path, json=group)
        if resp.status_code != 200:
            raise Exception("got status={} expect 200, body:\n{}".format(resp.status_code, resp.text))
        else:
            print("user {} is added to group {}".format(user_uuid, group_uuid))
    else:
        print("user {} already at group {}".format(user_uuid, group_uuid))
