import argparse
import importlib
import json
import os
from typing import List

import hvac
import requests
import time


class Vault:
    """The Vault class for Negentropy vault control."""

    def __init__(
            self,
            name: str,
            url: str,
            token: str,
            keys: List[str] = None,
    ):
        """Creates a new Vault instance.
        :param name: Original name of vault.
        :type name: str
        :param url: Base URL for the Vault instance being addressed.
        :type url: str
        :param token: Authentication token to include in requests sent to Vault.
        :type token: str
        :param keys: Shamir keys for unsealing vault
        :type keys: List[str]
        """
        self.name = name
        self.token = token
        self.url = url
        self.keys = keys
        self.vault_client = hvac.Client(url=self.url, token=token)

    def wait(self, attempts: int = 5, seconds_per_attempt: int = 1):
        """Wait for response of vault specified amount of time"""
        for i in range(attempts):
            try:
                self.vault_client.sys.is_initialized()
                return
            except Exception:
                time.sleep(seconds_per_attempt)
                continue
        raise Exception(
            "{} attempts were failed, vault '{}' at {} is unreachable".format(attempts, self.name, self.url))

    def init_and_unseal(self):
        """Init and unseal vault """
        print("Starting init and unseal vault '{}'".format(self.name))
        if self.vault_client.sys.is_initialized():
            print("vault '{}' already initialized, skip init".format(self.name))
            # it needs because of strange rejecting 'root' token when vault is dev mode
            if self.token == "root":
                print("vault '{}': creating new root token".format(self.name))
                resp = self.vault_client.auth.token.create(policies=['root'], no_parent=True)
                self.token = resp['auth']['client_token']
                print("vault '{}' new root_token: {}".format(self.name, self.token))
                self.vault_client = hvac.Client(url=self.url, token=self.token)
            else:
                file = open("/tmp/vaults", "r")
                content = file.read()
                file.close()
                cfgs = json.loads(content)
                for cfg in cfgs:
                    if cfg['name'] == self.name:
                        self.token = cfg['token']
                        self.keys = cfg['keys']
                        self.vault_client.token = self.token
        else:
            resp = self.vault_client.sys.initialize()
            self.keys = resp['keys_base64']
            print("vault '{}' shamir_keys: {}".format(self.name, self.keys))
            self.token = resp['root_token']
            print("vault '{}' root token: {}".format(self.name, self.token))
            self.vault_client.token = self.token
            if os.path.exists("/tmp/vaults") and not os.stat("/tmp/vaults").st_size == 0:
                file = open("/tmp/vaults", "r")
                content = file.read()
                file.close()
                cfgs = json.loads(content)
                idx = -1
                for i in range(0, len(cfgs)):
                    if cfgs[i]['name'] == self.name:
                        idx = i
                if idx > -1:
                    cfg = cfgs[idx]
                    cfg['url'] = self.url
                    cfg['token'] = self.token
                    cfg['keys'] = self.keys
                    file = open("/tmp/vaults", "r+")
                    file.seek(0)
                    json.dump(cfgs, file)
                    file.truncate()
                    file.close()
                else:
                    json_out = {"name": self.name, "token": self.token, "url": self.url, "keys": self.keys}
                    cfgs.append(json_out)
                    file = open("/tmp/vaults", "w")
                    file.write(json.dumps(cfgs))
                    file.close()
            else:
                json_out = [{"name": self.name, "token": self.token, "url": self.url, "keys": self.keys}]
                file = open("/tmp/vaults", "w")
                file.write(json.dumps(json_out))
                file.close()
        if not self.vault_client.sys.is_sealed():
            print("vault '{}' already unsealed, skip unseal".format(self.name))
        else:
            if self.keys is None:
                raise Exception("vault {}: empty shamir keys, stopped".format(self.name))
            self.vault_client.sys.submit_unseal_keys(keys=self.keys)
        print("vault '{}' successfully inited and unsealed".format(self.name))


def find_master_root_vault(vaults: List[Vault]) -> Vault:
    """ returns master root vault (now it is just first vault with flant_iam onboard) """
    for vault in vaults:
        if vault.name == "root":
            return vault
    raise Exception("there is no vaults with flant_iam onboard in passed: {}".format(vaults))


def check_response(resp: requests.Response, expected_status_code: int = 200) -> requests.Response:
    """ raise an exception if returned status code doesn't match expected"""
    if resp.status_code != expected_status_code:
        raise Exception(
            "expected {}, got {}, response json:\n {}".format(expected_status_code, resp.status_code, resp.text))
    return resp


def create_privileged_tenant(vault: Vault, tenant_uuid: str, identifier: str):
    """create tenant if not exists"""
    vault_client = hvac.Client(url=vault.url, token=vault.token)
    resp = vault_client.read(path="flant/tenant/" + tenant_uuid)
    if resp:
        print("tenant with uuid '{}' already exists".format(tenant_uuid))
        return
    vault_client.write(path="flant/tenant/privileged", uuid=tenant_uuid, identifier=identifier)
    print("tenant with uuid '{}' created".format(tenant_uuid))


proto_team_uuid = '58df57d6-d75b-4889-a1cf-15d95e90198a'


def create_privileged_team_proto(vault: Vault):
    """create proto team for first teammate"""
    vault_client = hvac.Client(url=vault.url, token=vault.token)
    resp = vault_client.read(path="flant/team/" + proto_team_uuid)
    if resp:
        print("proto team uuid '{}' already exists".format(proto_team_uuid))
        return
    vault_client.write(path='flant/team/privileged', uuid=proto_team_uuid, identifier="proto",
                       team_type='standard_team')
    print("proto team uuid  '{}' created".format(proto_team_uuid))


def create_privileged_teammate_to_proto_team(vault: Vault, teammate_uuid: str, teammate_email: str):
    """create user if not exists"""
    base_path = "flant/team/{}/teammate/".format(proto_team_uuid)
    vault_client = hvac.Client(url=vault.url, token=vault.token)
    resp = vault_client.read(path=base_path + teammate_uuid)
    if resp:
        print("teammate with uuid '{}' already exists".format(teammate_uuid))
        return
    vault_client.write(path=base_path + "privileged", uuid=teammate_uuid, identifier=teammate_email.split("@")[0],
                       email=teammate_email, role_at_team="member")
    print("teammate with uuid '{}' created".format(teammate_uuid))
    flant_tenant_uuid = "be0ba0d8-7be7-49c8-8609-c62ac1f14597"
    resp = vault_client.write(path=f"flant/tenant/{flant_tenant_uuid}/role_binding",
                              description=f"flant.admin for primary user: {teammate_email}",
                              members=[{
                                  "uuid": teammate_uuid,
                                  "type": "user",
                              }],
                              roles=[{
                                  "name": "flant.admin",
                                  "options": {},
                              }],
                              )
    print("'flant.admin' role for teammate with uuid '{}' is available".format(teammate_uuid))
    resp = vault_client.write(path=f"flant/tenant/{flant_tenant_uuid}/role_binding",
                              description=f"tenant.manage for primary user: {teammate_email}",
                              members=[{
                                  "uuid": teammate_uuid,
                                  "type": "user",
                              }],
                              roles=[{
                                  "name": "tenant.manage",
                                  "options": {},
                              }],
                              )
    print("'tenant.manage' role for teammate with uuid '{}' is available".format(teammate_uuid))


def create_privileged_user(vault: Vault, tenant_uuid: str, user_uuid: str, email: str):
    """create user if not exists"""
    base_path = "flant/tenant/{}/user/".format(tenant_uuid)
    vault_client = hvac.Client(url=vault.url, token=vault.token)
    resp = vault_client.read(path=base_path + user_uuid)
    if resp:
        print("user with uuid '{}' already exists".format(user_uuid))
        return
    vault_client.write(path=base_path + "privileged", uuid=user_uuid, identifier=email.split("@")[0], email=email)
    print("user with uuid '{}' created".format(user_uuid))


def create_user_multipass(vault: Vault, tenant_uuid: str, user_uuid: str, ttl_sec: int) -> str:
    """create user multipass"""
    vault_client = hvac.Client(url=vault.url, token=vault.token)
    resp = vault_client.write(path="flant/tenant/{}/user/{}/multipass".format(tenant_uuid, user_uuid), ttl=ttl_sec)
    body = resp.json()
    if type(body) is not dict:
        raise Exception("expect dict, got:{}".format(type(body)))
    if not body['data'] or \
            type(body['data']) is not dict \
            or not body['data']['token']:
        raise Exception("expect {'data':{'token':'xxxx', ...}, ...} dict, got:\n" + body)
    return body['data']['token']


def run_migrations(vaults: List[Vault]):
    module_path = './infra/common/docker/migrator.py'
    module_name = 'migrations'
    loader = importlib.machinery.SourceFileLoader(module_name, module_path)
    module = loader.load_module()
    migration_config = 'infra/common/config/environments/' + args.mode + '.yaml'
    migration_dir = 'infra/main/vault_migrations'
    module.upgrade_vaults([{'name': v.name, 'url': v.url, 'token': v.token} for v in vaults], migration_dir,
                          migration_config)


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


# Custom steps for single vault
def configure_ssh(vault: Vault):
    vault_client = hvac.Client(url=vault.url, token=vault.token)
    print("INFO: import ssh ca at '{}' vault".format(vault.name))
    vault_client.write(path='ssh/config/ca', private_key="""-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEA0/G1wVnF9ufvio1W1XBAD51EU6UP+p0otMVfpap/7DgkyZY0
WEzJNFGxmR271VdnnWGKYApAyjlhfXheYaY5j2rMmKLJFTCc/X2ntfnJfqZsnJxk
2S7KYNK+fTa/++68o2tipJZWOAl3O85Zrv0ft9elYM6Vj8keNNO5SGZdvAQGoW3X
yif4zaaZFWS+Nd60hWeYEwZTCFZmataVLzgbWoTKx9ig71nYNFCVoeao8h8Ynwvi
797x1pSqsC64CRUPOfVeLG306obeNV8LfNJ5CkgO8ji+BZ8RcMSauQ0iW+chk2J7
b902JcJpWZi9yYNeEt2kM1vNCG1bkcJw38L9JQIDAQABAoIBABSABaeNCmPmbToG
j8aXU+ruuEQq7A++kchiauz4P+VWTOCewbNkwfVojXgU8y0ghion3B2MAFZPFInx
UZe6X0jq+J0u6ao+CIFQXR9x6LZyXIENc4e6SeLxn3E3EXzJy782zNTEodRLvhev
zubpHt9GYX2qnbbJqj1L2VkSZbCgufku+4y4UbFINMImzwU9kZpc3rbqsYCSzNH+
x7cCsj1yuXK4Du+k5NX16jFnuZfES05h6Rq26egSkBSrhzTd8eP6YVun6JnJEVOw
vOqGyGVFMu5toOb8Wnjp5PEj6/c4oRzg+t1tXr1YUoo3RAA17JnqeHopVb8gz1d+
83bxpEUCgYEA8PiWUZ8Za++w1iE1XPt499504pwSzPTh5vbTl0nbE17YSfA0Dc4S
vyrZZLjmYKezqebM1Sw9/IJWblk4e6UbcRu0+XsQpeH2Yxv0h/fJTi43tYVyzSKP
70+IYJJBFJ3xfA8dPN8HqvkKUMHcQvdwU2DEC47wg15yrD0+sETF83cCgYEA4Smr
603VY5HB/Ic+ehAXMc/CFRB6bs2ytxJL254bmPWJablqHH25xYbe7weJEPGJedaw
Ek1r3hFjGddxLC4ix5i6YfH4NwRMBh0rU8YmAWHVyHVFlZecGTv+42dBxXzVxPS9
Hf/DFLy6r3L0FL+pcVxRy9Mm63e3ydnF54ptI0MCgYAHQDOluRfWu5uildU5Owfk
zXjO6MtYB3ZUsNClGL/S0WPItcWbNLwzrGJmOXoVJnatghhfwbkLxBA9ucmNTuaI
fMDxUNarZyU2zjyJatdP1uwuNhnCOmwCU25TGZODv0zo4ruKfVuJtXyt+WdbTH7A
w4SipGZwTYM904nzW95o+QKBgHRWmbO8xZLqzvZx0sAy7CkalcdYekoiEkMxOuzA
prXDuDpeSQtrkr8SzsFmfVW51zSSzurGAgP9q9zASoNvWx0SNstAwOV8XOOT0r04
Vo7ERDeNEGUYrtkC/NH2mi82LyXS5pxHeD6QvUzF8oN9/EjMUJ8l/KgRdW7gDLdz
+KwNAoGAQkNO/RWEsJYUkEUkkObfSqGN75s78fjT1yZ7CX0dUvHv6KC3+f7RmNHM
2zNxHZ+s+x9hfasJaduoV/hksluY4KUMuZjkfih8CaRIqCY8E/wEYjsyYJzJ4f1u
C+iz1LopgyIrKSebDzl13Yx9/J6dP3LrC+TiYyYl0bf4a4AStLw=
-----END RSA PRIVATE KEY-----""",
                       public_key='ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDT8bXBWcX25++KjVbVcEAPnURTpQ/6nSi0xV+lqn/sOCTJljRYTMk0UbGZHbvVV2edYYpgCkDKOWF9eF5hpjmPasyYoskVMJz9fae1+cl+pmycnGTZLspg0r59Nr/77ryja2KkllY4CXc7zlmu/R+316VgzpWPyR4007lIZl28BAahbdfKJ/jNppkVZL413rSFZ5gTBlMIVmZq1pUvOBtahMrH2KDvWdg0UJWh5qjyHxifC+Lv3vHWlKqwLrgJFQ859V4sbfTqht41Xwt80nkKSA7yOL4FnxFwxJq5DSJb5yGTYntv3TYlwmlZmL3Jg14S3aQzW80IbVuRwnDfwv0l'
                       )
    print("INFO: create ssh signer at '{}' vault".format(vault.name))
    vault_client.write(path='ssh/roles/signer', allow_user_certificates=True, algorithm_signer='rsa-sha2-256',
                       allowed_users='*', allowed_extensions='permit-pty,permit-agent-forwarding', key_type='ca',
                       default_extensions=[{"permit-agent-forwarding": "", "permit-pty": ""}], ttl='5m0s')


# Main scrypt block
if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', dest='mode')
    parser.add_argument('--okta-uuid', dest='okta_uuid')
    parser.add_argument('--okta-email', dest='okta_email')
    args = parser.parse_args()

    if args.mode == 'single':
        single_vault = Vault(name="root", url="http://127.0.0.1:8200", token="root")
        vaults = [single_vault]
    else:
        root_vault = Vault(name="root", url="http://127.0.0.1:8300", token="")
        auth_vault = Vault(name="auth", url="http://127.0.0.1:8200", token="")
        vaults = [root_vault, auth_vault]

    for vault in vaults:
        print("========================================")
        print("vault: {} at {}".format(vault.name, vault.url))
        print("========================================")
        vault.wait()
        vault.init_and_unseal()

    write_tokens_to_file(vaults)

    run_migrations(vaults)

    # ============================================================================
    # prepare user multipass_jwt for authd tests
    # ============================================================================
    multipass_file_path = "authd/dev/secret/authd.jwt"
    multipass_file_folder = multipass_file_path.rsplit("/", 1)[0]
    if not os.path.exists(multipass_file_folder):
        os.makedirs(multipass_file_folder)
    iam_vault = find_master_root_vault(vaults)
    create_privileged_tenant(iam_vault, "00000991-0000-4000-A000-000000000000", "tenant_for_authd_tests")
    create_privileged_user(iam_vault, "00000991-0000-4000-A000-000000000000",
                           "00000661-0000-4000-A000-000000000000",
                           "user_for_authd_tests@gmail.com")
    multipass = create_user_multipass(iam_vault, "00000991-0000-4000-A000-000000000000",
                                      "00000661-0000-4000-A000-000000000000", 3600)
    file = open(multipass_file_path, "w")
    file.write(multipass)
    file.close()
    print(multipass_file_path + " is updated")

    # ============================================================================
    # create teammate for webdev local development
    # ============================================================================

    if args.okta_uuid:
        print("DEBUG: OKTA UUID is", args.okta_uuid)
        print("DEBUG: OKTA EMAIL IS", args.okta_email)
        create_privileged_team_proto(root_vault)
        create_privileged_teammate_to_proto_team(root_vault, args.okta_uuid, args.okta_email)

    # single mode code run
    if len(vaults) < 2:
        configure_ssh(vaults[0])
