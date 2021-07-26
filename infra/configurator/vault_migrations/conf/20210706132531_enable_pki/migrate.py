"""
This module contains a vault migration.
Write your migration using hvac python module. See https://hvac.readthedocs.io/en/stable/overview.html for details.

Migration Name: enable_pki
Migration Version: 20210706132531
"""

from apply import *

def upgrade(connection):
    connection.sys.enable_secrets_engine(
        backend_type='pki',
        path='vault-cert-auth',
        max_lease_ttl='87600h',
    )

    generate_ca_response = connection.write(
        path='vault-cert-auth/root/generate/internal',
        common_name='negentropy',
        ttl='87600h'
    )
    generate_ca_response_data = generate_ca_response.get('data')
    ca = generate_ca_response_data['issuing_ca']
    upload_blob_from_string(terraform_state_bucket, str(ca), vault_auth_ca_name)

    connection.write(
        path='vault-cert-auth/roles/auth',
        allow_any_name='true',
        max_ttl='1h'
    )

    pass
