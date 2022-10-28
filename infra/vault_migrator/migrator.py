#!/usr/bin/env python3

import argparse
import ast
import glob
import importlib.machinery
import os
import base64

from typing import List, Callable
from dotenv import load_dotenv
import datetime

import sys

# TODO: vault secrets enable -path migrator database
from config import MIGRATION_TEMPLATE, log
from errors import Error

from utils import prepare_user_multipass_jwt, split_vaults, write_tokens_to_file, create_teammate_for_webdev, single_mode_code_run

from migration import Migration
from vault import Vault


def load_migrations(directory):
    """ Return the migrations contained in the given directory and sort then ascending. """
    if not os.path.exists(directory) and os.path.isdir(directory):
        msg = "%s is not a directory." % directory
        raise Error(msg)
    wildcard = os.path.join(directory, '*', 'migrate.py')
    migration_files = glob.glob(wildcard)
    migrations = [Migration(f) for f in migration_files]
    migrations.sort(key=lambda m: m.get_version())
    return migrations


def get_vault_version(url='', token=''):
    """ Return the migration version of the given vault. """
    db = Vault(url = url, token = token)
    return db.get_version()


def create_migration(name, directory=None):
    """ Create a migration with the given name. If no directory is specified,
        the current working directory will be used.
    """
    directory = directory if directory else '.'
    now = datetime.datetime.now()
    version = now.strftime("%Y%m%d%H%M%S")
    contents = MIGRATION_TEMPLATE % {'name': name, 'version': version}

    name = name.replace(' ', '_')
    migration_dirname = "%s_%s" % (version, name)
    path = os.path.join(directory, migration_dirname, 'migrate.py')

    os.makedirs(os.path.join(directory, migration_dirname), exist_ok=True)
    with open(path, 'w') as migration_file:
        migration_file.write(contents)
    return path

def create_migration_command(args):
    """creates a migration template"""
    name = args.name
    directory = args.migration_dir
    path = create_migration(name, directory)
    log.info("created migration %s" % path)

def print_status_command(args):
    
    vault_version = get_vault_version()

    msg = f'Vault version [{vault_version}]' if vault_version else 'the vault is not under version control'
    log.info(msg)
    
    migration_dir = args.migration_dir
    log.info(f"Migrations in [{migration_dir}]:")
    
    migrations = load_migrations(migration_dir)

    for migration in migrations:
        migration_version = migration.get_version()
        
        if vault_version >= migration_version:
            line = f"{migration_version}\t[applied]\t{migration.name}\t{migration.path}"
        else:
            line = "{migration_version}\t[not applied]\t{migration.name}\t{migration.path}"
        log.info(line)


def core_migrations_vault_filter(migration_type: str, all_vaults: List[Vault]) -> List[Vault]:
    """filter vaults by type of core_migration
    :param migration_type:
    :param all_vaults:
    :return:
    """
    if migration_type == 'all':
        return all_vaults
    return [v for v in all_vaults if  migration_type in v.name]


def is_migration_new(migration: Migration, vault: Vault) -> bool:
    """check is migration new for specified vault"""
    
    current_vault_version = vault.get_version()
    log.debug(f"current_version   [{current_vault_version}]")
    msg = f"migration version [{migration.get_version()}] for vault [{vault.name}]({vault.url})"
    if migration.get_version() > current_vault_version:
        log.info(msg + ' is new')
        return True
    else:
        log.debug(msg + ' is old')
        return False


def run_migration_at_vault(migration: Migration, vault: Vault, vaults: List[Vault]):
    """run passed migration for specified vault"""
    loader = importlib.machinery.SourceFileLoader('migration_' + migration.get_version(), migration.path)
    module = loader.load_module()
    module.upgrade(vault.name, vaults)


def upgrade_vaults(vaults: List[Vault], migration_dir: str, version: str = None):
    """
    operate migrations over given vaults
    :param vaults: example: [{'name': 'conf-conf', 'url': 'https://X.X.X.X:YYY', 'token': '...'}, {'name': 'auth-ew3a1', ...}, {'mane': 'root-source-3', ...}]
    :param migration_dir: one of '../../configurator/vault_migrations' or '../../main/vault_migrations'
    :param version: valid UTC timestamp, the last operated migration will not exceed, example: 20210716203309
    :return:
    """    
    core_vaults, other_vaults = split_vaults(vaults)
    
    # run core_migrations if core_vaults are passed
    if core_vaults:
        core_migrations = load_migrations(os.path.join(migration_dir, "core"))
    
        run_migrations(migrations=core_migrations, vaults=core_vaults, version=version,
                       vault_filter=core_migrations_vault_filter)
    
    # run other_migrations if core_vaults are passed
    if len(other_vaults) > 1:
        raise Error(f"allow only one not core vault, got '{len(other_vaults)}'")
    elif len(other_vaults) == 1:
        if 'conf-conf' in other_vaults[0].name:
            migration_dir = os.path.join(migration_dir, 'conf-conf')
        else:
            migration_dir = os.path.join(migration_dir, 'conf')
        not_core_migrations = load_migrations(migration_dir)
        run_migrations(migrations=not_core_migrations, vaults=other_vaults, version=version, vault_filter=None)


def run_migrations(migrations: List[Migration], vaults: List[Vault], vault_filter: Callable[[str, List[Vault]], List[Vault]] = None, version: str = None):
    """run all passed migration at vaults according passed vault_filter and version"""
    
    for m in migrations:
        if version and m.get_version() > version:
            break
        
        operate_vaults = vault_filter(migration_type=m.get_core_migration_type(), all_vaults=vaults) if vault_filter else vaults
        
        for v in operate_vaults:
            if is_migration_new(m, v):
                run_migration_at_vault(m, v, vaults)
                
                v.update_version(m.get_version())
                
                new_version = v.get_version()
                if new_version == m.get_version():
                    log.info(f"vault [{v.name}] upgraded successfully to version [{new_version}]")
                else:
                    log.info(f"vault [{v.name}] is NOT upgraded to version [{new_version}]")
                    exit(1)
 
 
def production(args):
    """ setup production environment

    Args:
        args: args 
    """
    # on production VAULTS_B64_JSON in base 64
    vaults_list = base64.b64decode(os.environ.get('VAULTS_B64_JSON'))
    if not vaults_list:
        log.info("The list of vaults is empty")
        exit()
    decoded_vaults_list = ast.literal_eval(base64.b64decode(vaults_list))
    
    # we need to create a list of Vaults from dicts to continue work with them
    vaults = [Vault(**v) for v in decoded_vaults_list]
    
    migration_dir = 'infra/vault_migrator/migrations'
    upgrade_vaults(vaults, migration_dir)

def local_env(args):
    """ setup local environment

    Args:
        args: args 
    """
    
    # loading needed environment variables from local .env
    load_dotenv()
   
    vaults_conf = ast.literal_eval(os.environ.get('VAULTS_B64_JSON'))
    
    # we need to create a list of Vaults from dicts to continue work with them
    vaults = [Vault(**v) for v in vaults_conf]
    
    for vault in vaults:
        print("========================================")
        print("vault: {} at {}".format(vault.name, vault.url))
        print("========================================")
        vault.wait(seconds_per_attempt=5)
        vault.init_and_unseal()
    
    write_tokens_to_file(vaults)
    
    migration_dir = 'infra/vault_migrator/migrations'
    upgrade_vaults(vaults, migration_dir)
    
    # customisation for local development
    
    root_vault = next(v for v in vaults if 'root' in v.name)
    
    prepare_user_multipass_jwt(root_vault)
    
    if args.okta_uuid:
        create_teammate_for_webdev(root_vault,args)

    # single mode code run
    if len(vaults) < 2:
        single_mode_code_run(vaults[0])


def main():
    parser = argparse.ArgumentParser()
    commands = parser.add_subparsers(help='commands')

    # add the create migration command
    create_cmd = commands.add_parser("create", help="create a new migration file")
    create_cmd.add_argument("--name",dest='name', help="the name of migration")
    create_cmd.add_argument("--migration_dir",dest='migration_dir', help="the migration directory")
    create_cmd.set_defaults(func=create_migration_command)

    # add the status command
    status_cmd = commands.add_parser("status", help="return status of vault and migrations")
    status_cmd.add_argument("migration_dir", help="the migration directory")
    status_cmd.set_defaults(func=print_status_command)
    
    
    # setup local environment
    local = commands.add_parser('local', help='create local environment')
    local.add_argument('--okta-uuid', dest='okta_uuid')
    local.add_argument('--okta-email', dest='okta_email')
    local.set_defaults(func=local_env)

    # setup prod environment
    local = commands.add_parser('production', help='create local environment')
    local.set_defaults(func=production)
    
    
    args = parser.parse_args()
    
    try:
        func = args.func
    except AttributeError:
        parser.error("too few arguments")
    func(args)


if __name__ == '__main__':
    sys.exit(main())
