from typing import Type, TypedDict, List


class Vault(TypedDict):
    name: str
    token: str
    url: str
    client: Type # hvac.Client()


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v.name == vault_name)

    print("INFO: import ssh ca at '{}' vault".format(vault_name))
    vault.client.write(path='ssh/config/ca', private_key="""-----BEGIN RSA PRIVATE KEY-----
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
    print("INFO: create ssh signer at '{}' vault".format(vault_name))
    vault.client.write(path='ssh/roles/signer', allow_user_certificates=True, algorithm_signer='rsa-sha2-256',
                       allowed_users='*', allowed_extensions='permit-pty,permit-agent-forwarding', key_type='ca',
                       default_extensions=[{"permit-agent-forwarding": "", "permit-pty": ""}], ttl='5m0s')
