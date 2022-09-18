import os
import json
import base64

print("Example migrate")

if __name__ == "__main__":
    vault_env = os.getenv("VAULTS_B64_JSON", default='W10=')
    vaults = json.loads(base64.b64decode(vault_env))
    for vault in vaults:
        print("Migrating vault", vault)