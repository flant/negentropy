from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


def upgrade(vault_name: str, vaults: List[Vault]):
    all_pubkeys = []
    print("INFO: get flant_iam_auth kafka public keys from all vaults")
    for vault in vaults:
        vault_client = hvac.Client(url=vault['url'], token=vault['token'])
        public_key = vault_client.read(path='auth/flant/kafka/public_key').get('data').get('public_key')
        all_pubkeys.append(public_key)
    print("INFO: get flant_iam kafka public key from 'root' vault")
    root_vault = next(v for v in vaults if 'root' in v['name'])
    root_vault_client = hvac.Client(url=root_vault['url'], token=root_vault['token'])
    public_key = root_vault_client.read(path='flant/kafka/public_key').get('data').get('public_key')
    all_pubkeys.append(public_key)
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: configure flant_iam_auth at vault: '{}'".format(vault_name))
    vault_client.write(path='auth/flant/kafka/configure', peers_public_keys=all_pubkeys,
                       self_topic_name='auth_source.' + vault_name, root_topic_name='root_source.' + vault_name,
                       root_public_key=public_key)
