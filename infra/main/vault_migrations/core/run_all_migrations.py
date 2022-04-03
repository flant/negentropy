# Run of all migrations for testing and debugging purposes

import os
import importlib

from typing import TypedDict, List
from setuptools import glob


class Vault(TypedDict):
    name: str
    token: str
    url: str


class Migration(TypedDict):
    path: str
    type: str
    stamp: int


def is_directory(path):
    return os.path.exists(path) and os.path.isdir(path)


def migration_vaults(migration_type: str, all_vaults: List[Vault]) -> List[Vault]:
    if migration_type == 'all':
        return all_vaults
    return [v for v in all_vaults if migration_type in v['name']]


def collect_and_sort_migrations(directory: str) -> List[Migration]:
    if not is_directory(directory):
        msg = "%s is not a directory." % directory
        raise Exception(msg)
    wildcard = os.path.join(directory, '*', 'migrate.py')
    migration_files = glob.glob(wildcard)
    migrations = []
    for path in migration_files:
        elements = path.split('/')
        dir = elements[-2]
        parts = dir.split('_')
        migrations.append(Migration(path=path, type=parts[1], stamp=int(parts[0])))
    migrations.sort(key=lambda m: m['stamp'])
    return migrations


def is_migration_new(migration: Migration, vault: Vault) -> bool:
    # TODO
    return True


def save_last_migration(migration: Migration, vault: Vault):
    pass


def run_migration_at_vault(migration: Migration, vault: Vault, vaults: List[Vault]):
    loader = importlib.machinery.SourceFileLoader('migration_' + str(migration['stamp']), migration['path'])
    module = loader.load_module()
    module.upgrade(vault['name'], vaults)


def run_all_migrations(vaults, directory):
    print("It is all migrations run!")
    print(vaults)
    migrations = collect_and_sort_migrations(directory)
    print(migrations)
    for m in migrations:
        operate_vaults = migration_vaults(migration_type=m['type'], all_vaults=vaults)
        for v in operate_vaults:
            if is_migration_new(m, v):
                run_migration_at_vault(m, v, vaults)
                save_last_migration(m, v)
    print("It is all migrations run ends!")

# vaults = [{'name': 'root', 'url': 'http://127.0.0.1:8300', 'token': 's.qoeHXBNTscgZXYprEsNcCAAZ'},
#           {'name': 'auth', 'url': 'http://127.0.0.1:8200', 'token': 's.LE3lTWROmvrjS4ed2RhLhvok'}]
#
#
# directory = '.'
# run_all_migrations(vaults, directory)
