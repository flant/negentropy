from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


def upgrade(vault_name: str, vaults: List[Vault]):
    all_pubkeys = []
    print("INFO: get flant_iam_auth kafka public keys from all vaults")
    for vault in vaults:
        public_key = vault.client.read(path='auth/flant/kafka/public_key').get('data').get('public_key')
        all_pubkeys.append({'name': vault.name, 'public_key': public_key})
    print("INFO: configure flant_iam replicas at 'root' vault")
    root_vault = next(v for v in vaults if 'root' in v.name)

    # TODO: check existing replicas and add a new one to them
    for key in all_pubkeys:
        root_vault.client.write(path='flant/replica/' + key['name'],
        type='Vault', public_key=key['public_key'])
