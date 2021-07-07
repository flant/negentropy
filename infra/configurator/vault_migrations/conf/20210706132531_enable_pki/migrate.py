"""
This module contains a vault migration.
Write your migration using hvac python module. See https://hvac.readthedocs.io/en/stable/overview.html for details.

Migration Name: enable_pki
Migration Version: 20210706132531
"""
import hvac

def upgrade(connection):
    # add your upgrade step here
    # connection is an instance of hvac.Client()
    # example code:
    # print(connection.is_authenticated())
    #
    connection.sys.enable_secrets_engine(
        backend_type='pki',
        path='pki',
        max_lease_ttl='87600h',
    )
    # connection.sys.tune_mount_configuration(
    #         path='pki',
    #         max_lease_ttl='87600h',
    # )
    pass
