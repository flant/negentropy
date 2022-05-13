from typing import TypedDict, List

import hvac


class Vault(TypedDict):
    name: str
    token: str
    url: str


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    print("INFO: configure flant_iam_auth at vault: '{}'".format(vault_name))
    public_key = vault_client.read(path='flant/kafka/public_key').get('data').get('public_key')
    vault_client.write(path='auth/flant/kafka/configure', peers_public_keys=public_key,
                       self_topic_name='auth_source.' + vault_name, root_topic_name='root_source.' + vault_name,
                       root_public_key=public_key)
