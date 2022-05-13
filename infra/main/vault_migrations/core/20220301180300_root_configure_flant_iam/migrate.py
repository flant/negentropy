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
    print("INFO: configure flant_iam at 'root' vault")
    root_vault = next(v for v in vaults if 'root' in v['name'])
    root_vault_client = hvac.Client(url=root_vault['url'], token=root_vault['token'])
    # TODO: check existing public keys and add a new one to them
    root_vault_client.write(path='flant/kafka/configure', self_topic_name="root_source",
                            peers_public_keys=all_pubkeys)
