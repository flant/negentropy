import json
import os
import time
from typing import Dict, List

import hvac

from config import VERSION_KEY, log


class Vault:
    """The Vault class for Negentropy vault control."""

    def __init__(self, name: str, url: str = '', token: str = '', keys: List[str] = None , vault_cacert: str = ''):
        """Creates a new Vault instance.
        :param name: Original name of vault. Eg: auth-2
        :type name: str
        :param url: Base URL for the Vault instance being addressed.
        :type url: str
        :param token: Authentication token to include in requests sent to Vault.
        :type token: str
        :param keys: Shamir keys for unsealing vault
        :type keys: List[str]
        """
        self.url = url if url else os.getenv('VAULT_ADDR') 
        self.token = token if token else os.getenv('VAULT_TOKEN')
        self.name = name
        self.keys = keys
        self.client = hvac.Client(url=self.url, token=self.token)

        log.debug(f"Vault {self.name} with token - {self.token} and url - {self.url} initialized")

    def wait(self, attempts: int = 5, seconds_per_attempt: int = 1):
        """Wait for response of vault specified amount of time"""
        for i in range(attempts):
            try:
                self.client.sys.is_initialized()
                return
            except Exception:
                time.sleep(seconds_per_attempt)
                continue
        raise Exception(f"{attempts} attempts were failed, vault '{self.name}' at {self.url} is unreachable")

    def init_and_unseal(self):
        """Init and unseal vault """
        log.info(f"Starting init and unseal vault '{self.name}'")

        if self.client.sys.is_initialized():
            log.info(f"vault '{self.name}' already initialized, skip init")
            # it needs because of strange rejecting 'root' token when vault is dev mode
            if self.token == "root":
                log.info(f"vault '{self.name}': creating new root token")
                resp = self.client.auth.token.create(policies=['root'], no_parent=True)
                self.token = resp['auth']['client_token']

                log.info(f"vault '{self.name}' new root_token: {self.token}")
                self.client = hvac.Client(url=self.url, token=self.token)
            else:
                file = open("/tmp/vaults", "r")
                content = file.read()
                file.close()
                cfgs = json.loads(content)
                for cfg in cfgs:
                    if cfg['name'] == self.name:
                        self.token = cfg['token']
                        self.keys = cfg['keys']
                        self.client.token = self.token
        else:
            resp = self.client.sys.initialize()
            self.keys = resp['keys_base64']
            
            log.info(f"vault '{self.name}' shamir_keys: {self.keys}")

            self.token = resp['root_token']

            log.info(f"vault '{self.name}' root token: {self.token}")
            
            self.client.token = self.token
            
            if os.path.exists("/tmp/vaults") and not os.stat("/tmp/vaults").st_size == 0:
                file = open("/tmp/vaults", "r")
                content = file.read()
                file.close()
                cfgs = json.loads(content)
                
                vault_cfg = next((cfg for cfg in cfgs if cfg['name'] == self.name), None)
                if vault_cfg:
                    vault_cfg['url'] = self.url
                    vault_cfg['token'] = self.token
                    vault_cfg['keys'] = self.keys
                    file = open("/tmp/vaults", "r+")
                    
                    file.seek(0)
                    json.dump(cfgs, file)
                    file.truncate()
                    file.close()
                else:
                    cfgs.append(self.toDict())
                    file = open("/tmp/vaults", "w")
                    file.write(json.dumps(cfgs))
                    file.close()
            else:
                json_out = [self.toDict()]
                file = open("/tmp/vaults", "w")
                file.write(json.dumps(json_out))
                file.close()
                
        if not self.client.sys.is_sealed():
            log.info(f"vault '{self.name}' already unsealed, skip unseal")
        else:
            if not self.keys:
                raise Exception(f"vault {self.name}: empty shamir keys, stopped")
            self.client.sys.submit_unseal_keys(keys=self.keys)
        log.info(f"vault '{self.name}' successfully inited and unsealed")
    
    def is_version_controlled(self):
        try:
            ret = self.client.secrets.kv.read_secret_version(path=VERSION_KEY)
        except:
            ret = False
        return bool(ret)
    
    def get_version(self):
        """ Return the vault's version, or None if it is not under version
            control.
        """
        if not self.is_version_controlled():
            self.initialize_version_control()
        result = self.client.secrets.kv.read_secret_version(path=VERSION_KEY)
        return result['data']['data']['version'] if result else "0"
    
    def update_version(self, version):
        if not self.is_version_controlled():
            self.initialize_version_control()
        self.client.secrets.kv.create_or_update_secret(path=VERSION_KEY, secret=dict(version=version))
    
    def initialize_version_control(self):
        list_mounted_secrets_engines = self.client.sys.list_mounted_secrets_engines().keys()
        
        if not 'secret/' in list_mounted_secrets_engines:
            self.client.sys.enable_secrets_engine(
                backend_type='kv',
                path='secret',
                options={'version': 1},
            )
        self.client.secrets.kv.create_or_update_secret(path=VERSION_KEY, secret=dict(version='0'))
    
    def create_privileged_tenant(self, tenant_uuid: str, identifier: str):
        """create privileged tenant if not exists(for local usage only)"""
        if self.client.read(path="flant/tenant/" + tenant_uuid):
            log.info(f"tenant with uuid '{tenant_uuid}' already exists")
            return
        self.client.write(path="flant/tenant/privileged", uuid=tenant_uuid, identifier=identifier)
        log.info(f"tenant with uuid '{tenant_uuid}' created")

    def toDict(self) -> Dict:
        """Vault dict representation

        Returns:
            _type_: {"name": self.name, "token": self.token, "url": self.url, "keys": self.keys}
        """
        return {"name": self.name, "token": self.token, "url": self.url, "keys": self.keys}
    
    def create_privileged_user(self, tenant_uuid: str, user_uuid: str, email: str):
        """create privileged user if not exists(for local usage only)"""
        base_path = f"flant/tenant/{tenant_uuid}/user/"
        
        resp = self.client.read(path=base_path + user_uuid)
        if resp:
            log.info(f"user with uuid '{user_uuid}' already exists")
            return
        self.client.write(path=base_path + "privileged", uuid=user_uuid, identifier=email.split("@")[0], email=email)
        log.info(f"user with uuid '{user_uuid}' created")
        
    def create_user_multipass(self, tenant_uuid: str, user_uuid: str, ttl_sec: int) -> str:
        """create user multipass (for local usage only)"""
    
        resp = self.client.write(path=f"flant/tenant/{tenant_uuid}/user/{user_uuid}/multipass", ttl=ttl_sec)
        body = resp.json()
        
        if type(body) is not dict:
            raise Exception(f"expect dict, got:{type(body)}")
        if not body['data'] or type(body['data']) is not dict or not body['data']['token']:
            raise Exception("expect {'data':{'token':'xxxx', ...}, ...} dict, got:\n" + body)
        return body['data']['token']    
        
    def __repr__(self):
        return 'Vault()'
    