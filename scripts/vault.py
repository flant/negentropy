import hvac
import time
import requests
from  json import dumps

from hvac import exceptions

from consts import negentropy_plugins, FLANT_IAM_AUTH

auth_plugins = {"flant_iam_auth"}


def check_response(resp: requests.Response, expected_status_code: int = 200) -> requests.Response:
    """ raise an exception if returned status code doesn't match expected"""
    if resp.status_code != expected_status_code:
        raise Exception(
            "expected {}, got {}, response json:\n {}".format(expected_status_code, resp.status_code, resp.text))
    return resp


class Vault:
    """The Vault class for Negentropy vault control."""

    def __init__(
            self,
            name: str,
            url: str,
            token: str,
            plugin_names: list[str],
            keys: list[str] = None,
    ):
        """Creates a new Vault instance.
        :param name: original name of vault.
        :type name: str
        :param url: Base URL for the Vault instance being addressed.
        :type url: str
        :param token: Authentication token to include in requests sent to Vault.
        :type token: str
        :param plugin_names: Valid names of negentropy plugins
        :type plugin_names: list[str]
        :param keys: shamir keys for unsealing vault
        :type keys: list[str]
        """
        self.name = name
        self.token = token
        self.url = url
        self.plugin_names = plugin_names
        self.keys = keys
        self.vault_client = hvac.Client(url=self.url, token=token)
        self.request_log = []

    def wait(self, attempts: int = 5, seconds_per_attempt: int = 1):
        """ wait for response of vault specified amount of time"""
        for i in range(attempts):
            try:
                self.vault_client.sys.is_initialized()
                return
            except Exception:
                time.sleep(seconds_per_attempt)
                continue
        msg = "{} attempts were falied, vault '{}' at {} is unreachable".format(attempts, self.name, self.url)
        raise Exception(msg)

    def init_and_unseal(self):
        """ init and unseal vault """
        print("run init_and_unseal for '{}' vault".format(self.name))
        if self.vault_client.seal_status['initialized']:
            print("vault '{}' already initialized, skip init".format(self.name))
        else:
            resp = self.vault_client.sys.initialize()
            self.keys = resp['keys_base64']
            print("vault '{}' new shamir_keys: {}".format(self.name, self.keys))
            self.token = resp['root_token']
            print("vault '{}' new root_token: {}".format(self.name, self.token))
            self.vault_client.token = self.token
        self.unseal()
        print("init_and_unseal for '{}' vault succeed".format(self.name))

    def unseal(self):
        """ unseal vault """
        print("run unseal for '{}' vault ".format(self.name))
        if not self.vault_client.seal_status['sealed']:
            print("vault '{}' already unsealed, skip unseal".format(self.name))
            return
        if self.keys is None:
            raise Exception("vault {}: empty shamir keys, stopped".format(self.name))
        self.vault_client.sys.submit_unseal_keys(keys=self.keys)
        if self.vault_client.seal_status['sealed']:
            raise Exception("vault {}: not unsealed, stopped".format(self.name))
        print("unseal for '{}' vault succeed".format(self.name))

    def activate_plugins(self):
        """ activate plugins """
        print("run activate_plugins for '{}' vault ".format(self.name))
        auths = set(self.vault_client.sys.list_auth_methods().keys())
        secrets = set(self.vault_client.sys.list_mounted_secrets_engines().keys())
        active_plugins = {name.removesuffix("/") for name in auths.union(secrets)}
        for p in self.plugin_names:
            if p in active_plugins:
                print("plugin '{}' already activated at '{}' vault".format(p, self.name))
            else:
                if p in auth_plugins:
                    self.vault_client.sys.enable_auth_method(
                        method_type=p,
                        path=p,
                        plugin_name=p,
                    )
                else:
                    self.vault_client.sys.enable_secrets_engine(
                        backend_type=p,
                        path=p,
                        plugin_name=p,
                    )
                print("plugin '{}' is activated at '{}' vault".format(p, self.name))

    def write_to_plugin(self, plugin: str, path: str, json: dict = None) -> requests.Response:
        """ write to plugin """
        return self.request_to_plugin(plugin, path, json, "POST")

    def put_to_plugin(self, plugin: str, path: str, json: dict = None) -> requests.Response:
        """ write to plugin """
        return self.request_to_plugin(plugin, path, json, "PUT")

    def read_from_plugin(self, plugin: str, path: str, json: dict = None) -> requests.Response:
        """ read from plugin """
        return self.request_to_plugin(plugin, path, json, "GET")

    def request_to_plugin(self, plugin: str, path: str, json: dict = None, method: str = "GET") -> requests.Response:
        """ request to plugin, check is plugin at vault """
        if plugin not in self.plugin_names:
            raise Exception("vault '{}': has not '{}' plugin".format(self.name, plugin))
        url = self.vault_client.url + "/v1" + \
              ("/auth/" if plugin in auth_plugins else "/") + \
              plugin + "/" + path.removeprefix(plugin)

        payload = dumps(json)
        headers = {
            'X-Vault-Token': self.token,
            'Content-Type': 'application/json',
        }
        request_timestamp = time.time()
        response = requests.request(method, url, headers=headers, data=payload)

        self.request_log.append({"request": {
            "timestamp": request_timestamp,
            "method": method,
            "url": url,
            "payload": payload},
            "response": response})

        return response

    def get_request_log(self):
        return self.request_log

    def activate_plugins_jwt(self):
        print("enable jwt at plugins at vault: '{}'".format(self.name))
        for plugin_name in self.plugin_names:
            if plugin_name in negentropy_plugins:
                print("\tplugin:{}".format(plugin_name))
                check_response(
                    self.write_to_plugin(
                        plugin=plugin_name,
                        path="jwt/enable?force=true"
                    ))

    def configure_self_access_for_flant_iam_auth(self):
        pass
        # create full policy
        print("create/update full policy")
        check_response(
            self.vault_client.sys.create_or_update_policy(name="full",
                                                          policy='path "*" {capabilities = ["create", "read", "update", "delete", "list"]}'
                                                          ), 204)
        # approle, secretID & roleID
        print("enable approle/role.full, getting secret_is and role_id")
        self.enable_approle()
        check_response(
            self.vault_client.auth.approle.create_or_update_approle(role_name="full", mount_point="approle",
                                                                    secret_id_ttl="5m", token_ttl="120s",
                                                                    token_policies=["full"]), 204)
        role_id = self.vault_client.auth.approle.read_role_id(role_name="full", mount_point="approle").get("data").get(
            "role_id")
        secret_id = self.vault_client.auth.approle.generate_secret_id(role_name="full", mount_point="approle").get(
            "data").get("secret_id")
        # configure access
        print("writing  auth/flant_iam_auth/configure_vault_access")
        if FLANT_IAM_AUTH in self.plugin_names:
            check_response(
                self.write_to_plugin(plugin=FLANT_IAM_AUTH, path="configure_vault_access", json={
                    "vault_addr": "http://127.0.0.1:8200",
                    "vault_tls_server_name": "vault_host",
                    "role_name": "full",
                    "secret_id_ttl": "5m",
                    "approle_mount_point": "/auth/approle/",
                    "role_id": role_id,
                    "secret_id": secret_id,
                    "vault_api_ca": "",
                }))

    def enable_approle(self):
        auth_methods = self.vault_client.sys.list_auth_methods()
        if "approle/" not in auth_methods:
            check_response(self.vault_client.sys.enable_auth_method(method_type="approle"), 204)

    def activate_auth_multipass(self):
        if FLANT_IAM_AUTH in self.plugin_names:
            print("writing auth/flant_iam_auth/auth_method/multipass")
            check_response(
                self.write_to_plugin(plugin=FLANT_IAM_AUTH, path="auth_method/multipass", json={
                    "token_ttl": "30m",
                    "token_policies": "full",
                    "token_no_default_policy": True,
                    "method_type": "multipass_jwt"
                }))

    def configure_ssh_ca(self, vault_names: list[str]):
        if self.name in vault_names and "ssh" in self.plugin_names:
            print("writing ssh/config/ca")
            try:
                self.vault_client.adapter.get(url="v1/ssh/config/ca").get("data").get("public_key")
            except (NameError, exceptions.InvalidRequest):
                check_response(
                    self.vault_client.adapter.put(url="v1/ssh/config/ca", json={
                        "private_key": """-----BEGIN RSA PRIVATE KEY-----
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
                        "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDT8bXBWcX25++KjVbVcEAPnURTpQ/6nSi0xV+lqn/sOCTJljRYTMk0UbGZHbvVV2edYYpgCkDKOWF9eF5hpjmPasyYoskVMJz9fae1+cl+pmycnGTZLspg0r59Nr/77ryja2KkllY4CXc7zlmu/R+316VgzpWPyR4007lIZl28BAahbdfKJ/jNppkVZL413rSFZ5gTBlMIVmZq1pUvOBtahMrH2KDvWdg0UJWh5qjyHxifC+Lv3vHWlKqwLrgJFQ859V4sbfTqht41Xwt80nkKSA7yOL4FnxFwxJq5DSJb5yGTYntv3TYlwmlZmL3Jg14S3aQzW80IbVuRwnDfwv0l"
                    }), 204)
            print("writing ssh/roles/signer")
            check_response(
                self.vault_client.adapter.post(url="v1/ssh/roles/signer", json={
                    "allow_user_certificates": True,
                    "algorithm_signer": "rsa-sha2-256",
                    "allowed_users": "*",
                    "allowed_extensions": "permit-pty,permit-agent-forwarding",
                    "key_type": "ca",
                    "default_extensions": {"permit-agent-forwarding": "", "permit-pty": ""},
                    "ttl": "5m0s"
                }), 204)

# "default_extensions": "{\"permit-agent-forwarding\": \"\",\"permit-pty\":\"\"}",
# "default_extensions": [{"permit-agent-forwarding": "11221", "permit-pty": "11111"}],
