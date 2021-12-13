import hvac
import time
import requests
import json

auth_plugins = {"flant_iam_auth"}

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

    def write_to_plugin(self, plugin: str, path: str, body: dict = None) -> requests.Response:
        """ write to plugin """
        return self.request_to_plugin(plugin, path, body, "POST")

    def read_from_plugin(self, plugin: str, path: str, body: dict = None) -> requests.Response:
        """ read from plugin """
        return self.request_to_plugin(plugin, path, body, "GET")

    def request_to_plugin(self, plugin: str, path: str, body: dict = None, method: str = "GET") -> requests.Response:
        """ request to plugin, check is plugin at vault """
        if plugin not in self.plugin_names:
            raise Exception("vault '{}': has not '{}' plugin".format(self.name, plugin))
        url = self.vault_client.url + "/v1" + \
              ("/auth/" if plugin in auth_plugins else "/") + \
              plugin + "/" + path.removeprefix(plugin)

        payload = json.dumps(body)
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

    def request_log(self):
        return self.request_log
