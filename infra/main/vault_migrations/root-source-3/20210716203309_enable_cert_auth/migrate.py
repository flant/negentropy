"""
This module contains a vault migration.
Write your migration using hvac python module. See https://hvac.readthedocs.io/en/stable/overview.html for details.
Migration Name: enable_cert_auth
Migration Version: 20210716203309
"""

def upgrade(connection):
    connection.sys.enable_auth_method(method_type='cert')
    pass
