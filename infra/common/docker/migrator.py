import argparse
import datetime
import glob
import importlib.machinery
import os
import sys
import traceback
from typing import TypedDict, List, Callable

import hvac

# TODO
# vault secrets enable -path migrator database
VERSION_KEY = 'migratorversion'
UTC_LENGTH = 14


class Error(Exception):
    """ Base class for all errors. """
    pass


class InvalidMigrationError(Error):
    """ Migration contains an error. """
    pass


class InvalidNameError(Error):
    """ Migration has an invalid filename. """

    def __init__(self, filename):
        msg = 'Migration filenames must start with a UTC timestamp. ' \
              'The following file has an invalid name: %s' % filename
        super(InvalidNameError, self).__init__(msg)


# code

def has_method(an_object, method_name):
    return hasattr(an_object, method_name) and \
           callable(getattr(an_object, method_name))


def is_directory(path):
    return os.path.exists(path) and os.path.isdir(path)


class Migration(object):
    """ This class represents a migration version. """

    def __init__(self, path):
        self.path = path  # ../../configurator/vault_migrations/conf/20210706132531_enable_pki/migrate.py
        self.filename = os.path.basename(os.path.dirname(path))  # 20210706132531_enable_pki
        self.dirname = os.path.basename(os.path.dirname(path))  # 20210706132531_enable_pki
        self.module_name, _ = os.path.splitext(self.filename)  # 20210706132531_enable_pki
        self.get_version()  # will assert the filename is valid
        self.name = self.module_name[UTC_LENGTH:]  # enable_pki
        while self.name.startswith('_'):
            self.name = self.name[1:]
        try:
            loader = importlib.machinery.SourceFileLoader(self.module_name, path)
            self.module = loader.load_module()
        except:
            msg = "Invalid migration %s: %s" % (path, traceback.format_exc())
            raise InvalidMigrationError(msg)
        # assert the migration has the needed methods
        missing = [m for m in ['upgrade']
                   if not has_method(self.module, m)]
        if missing:
            msg = 'Migration %s is missing required methods: %s.' % (
                self.path, ', '.join(missing))
            raise InvalidMigrationError(msg)

    def get_version(self):
        if len(self.dirname) < UTC_LENGTH:
            raise InvalidNameError(self.filename)
        timestamp = self.dirname[:UTC_LENGTH]
        return timestamp

    def upgrade(self, conn):
        self.module.upgrade(conn)

    def get_core_migration_type(self) -> str:
        """
        return type of core migration, if applicable
        :return:
        """
        parts = self.name.split("_")
        core_migration_type = parts[0]
        if core_migration_type not in ['auth', 'root', 'all']:
            raise Error('wrong type: %s of migration: %s' % (core_migration_type, self.name))
        return core_migration_type

    def __repr__(self):
        return 'Migration(%s)' % self.filename


class Vault(object):

    def __init__(self, url='', token=''):
        if url == '':
            url = os.getenv('VAULT_ADDR')
        if token == '':
            token = os.getenv('VAULT_TOKEN')
        self.conn = hvac.Client(
            url=url,
            token=token,
        )

    def is_version_controlled(self):
        try:
            ret = self.conn.secrets.kv.read_secret_version(path=VERSION_KEY)
        except:
            ret = False
        return bool(ret)

    # def upgrade(self, migrations, target_version=None):
    #     if target_version:
    #         _assert_migration_exists(migrations, target_version)
    #
    #     migrations.sort(key=lambda x: x.get_version())
    #     vault_version = self.get_version()
    #     for migration in migrations:
    #         current_version = migration.get_version()
    #         if current_version <= vault_version:
    #             continue
    #         if target_version and current_version > target_version:
    #             break
    #         Console.info("")
    #         Console.info(">>>>>> starting migration %s" % migration.get_version())
    #         migration.upgrade(self.conn)
    #         new_version = migration.get_version()
    #         self.update_version(new_version)

    def get_version(self):
        """ Return the vault's version, or None if it is not under version
            control.
        """
        if not self.is_version_controlled():
            self.initialize_version_control()
        result = self.conn.secrets.kv.read_secret_version(path=VERSION_KEY)
        return result['data']['data']['version'] if result else "0"

    def update_version(self, version):
        if not self.is_version_controlled():
            self.initialize_version_control()
        self.conn.secrets.kv.create_or_update_secret(path=VERSION_KEY, secret=dict(version=version))

    def initialize_version_control(self):
        list_mounted_secrets_engines = self.conn.sys.list_mounted_secrets_engines().keys()
        if not 'secret/' in list_mounted_secrets_engines:
            self.conn.sys.enable_secrets_engine(
                backend_type='kv',
                path='secret',
                options={'version': 1},
            )
        self.conn.secrets.kv.create_or_update_secret(path=VERSION_KEY, secret=dict(version='0'))

    def __repr__(self):
        return 'Vault()'


def load_migrations(directory):
    """ Return the migrations contained in the given directory. """
    if not is_directory(directory):
        msg = "%s is not a directory." % directory
        raise Error(msg)
    wildcard = os.path.join(directory, '*', 'migrate.py')
    migration_files = glob.glob(wildcard)
    return [Migration(f) for f in migration_files]


def get_vault_version(url='', token=''):
    """ Return the migration version of the given vault. """
    db = Vault(url, token)
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


MIGRATION_TEMPLATE = """\
\"\"\"
This module contains a vault migration.
Write your migration using hvac python module. See https://hvac.readthedocs.io/en/stable/overview.html for details.
Migration should be idempotent, if repeated write is wrong operation, use read before write 


Migration Name: %(name)s
Migration Version: %(version)s
\"\"\"

def upgrade(connection):
    # add your upgrade step here
    # connection is an instance of hvac.Client()
    # example code:
    # print(connection.is_authenticated())
    #
    pass
"""


class Console(object):

    @staticmethod
    def error(message):
        sys.stderr.write('%s\n' % message)

    @staticmethod
    def info(message):
        sys.stdout.write('%s\n' % message)


def create_migration_command(args):
    name = args.name
    directory = args.migration_dir
    path = create_migration(name, directory)
    Console.info("created migration %s" % path)


def print_status_command(args):
    vault_version = get_vault_version()
    msg = 'the vault is not under version control'
    if vault_version:
        msg = 'Vault version            [%s]' % (vault_version)
    Console.info(msg)
    Console.info("")

    migration_dir = args.migration_dir
    Console.info("Migrations in          [%s]:" % migration_dir)
    Console.info("")
    migrations = load_migrations(migration_dir)

    for migration in migrations:
        migration_version = migration.get_version()
        path = migration.path
        name = migration.name
        if vault_version >= migration_version:
            line = "%s\t%s\t%s\t%s" % (migration_version, "[applied]", name, path)
        else:
            line = "%s\t%s\t%s\t%s" % (migration_version, "[not applied]", name, path)
        Console.info(line)


def collect_and_sort_migrations(directory: str) -> List[Migration]:
    """collect migrations from specified path and sort then ascending"""
    if not is_directory(directory):
        msg = "%s is not a directory." % directory
        raise Error(msg)
    wildcard = os.path.join(directory, '*', 'migrate.py')
    migration_files = glob.glob(wildcard)
    migrations = []
    for path in migration_files:
        migrations.append(Migration(path=path))
    migrations.sort(key=lambda m: m.get_version())
    return migrations


class VaultParams(TypedDict):
    """vault connection params"""
    name: str
    token: str
    url: str


def core_migrations_vault_filter(migration_type: str, all_vaults: List[VaultParams]) -> List[VaultParams]:
    """
    filter vaults by type of core_migration
    :param migration_type:
    :param all_vaults:
    :return:
    """
    if migration_type == 'all':
        return all_vaults
    return [v for v in all_vaults if migration_type in v['name']]


def split_vaults(vaults: List[VaultParams]) -> (List[VaultParams], List[VaultParams]):
    """
    splits passed vaults into two groups
    :param vaults:
    :return: two lists of vaults: first list contains auth and root-source vaults, second one - others
    """
    core_vaults = []
    other_vaults = []
    for v in vaults:
        if 'auth' in v['name'] or 'root' in v['name']:
            core_vaults.append(v)
        else:
            other_vaults.append(v)
    return core_vaults, other_vaults


def is_migration_new(migration: Migration, vault: VaultParams) -> bool:
    """check is migration new for specified vault"""
    current_vault_version = get_vault_version(url=vault.get('url'), token=vault.get('token'))
    Console.info('current_version   [%s]' % current_vault_version)
    msg = 'migration version [%s] for vault [%s]' % (migration.get_version(), vault.get('name'))
    if migration.get_version() > current_vault_version:
        msg += ' is new'
        Console.info(msg)
        return True
    else:
        msg += ' is old'
        Console.info(msg)
        return False


def update_migration(migration: Migration, vault: VaultParams):
    """store new version in vault"""
    vault = Vault(url=vault.get('url'), token=vault.get('token'))
    vault.update_version(migration.get_version())


def run_migration_at_vault(migration: Migration, vault: VaultParams, vaults: List[VaultParams]):
    """run passed migration for specified vault"""
    loader = importlib.machinery.SourceFileLoader('migration_' + migration.get_version(), migration.path)
    module = loader.load_module()
    module.upgrade(vault['name'], vaults)


def upgrade_vaults(vaults: List[VaultParams], migration_dir: str, version: str = None):
    """
    operate migrations over given vaults
    :param vaults: example: [{'name': 'conf-conf', 'url': 'https://X.X.X.X:YYY', 'token': '...'}, {'name': 'auth-ew3a1', ...}, {'name': 'root-source-3', ...}]
    :param migration_dir: one of '../../configurator/vault_migrations' or '../../main/vault_migrations'
    :param version: valid UTC timestamp, the last operated migration will not exceed, example: 20210716203309
    :return:
    """
    print("upgrade_vaults run")
    core_vaults, other_vaults = split_vaults(vaults)
    if len(other_vaults) > 1:
        raise Error("allow only one not core vault, got '%s'" % other_vaults)
    # run core_migrations if core_vaults are passed
    if len(core_vaults) > 0:
        core_migrations = collect_and_sort_migrations(os.path.join(migration_dir, "core"))
        run_migrations(migrations=core_migrations, vaults=core_vaults, version=version,
                       vault_filter=core_migrations_vault_filter)
    # run other_migrations if core_vaults are passed
    if len(other_vaults) > 0:
        if 'conf-conf' in other_vaults[0].get('name'):
            migration_dir = os.path.join(migration_dir, 'conf-conf')
        else:
            migration_dir = os.path.join(migration_dir, 'conf')
        not_core_migrations = collect_and_sort_migrations(migration_dir)
        run_migrations(migrations=not_core_migrations, vaults=other_vaults, version=version, vault_filter=None)


def run_migrations(migrations: List[Migration], vaults: List[VaultParams],
                   vault_filter: Callable[[str, List[VaultParams]], List[VaultParams]] = None, version: str = None):
    """run all paased migration at vaults according passed vault_fliter and version"""
    for m in migrations:
        if version and m.get_version() > version:
            break
        if vault_filter:
            operate_vaults = vault_filter(migration_type=m.get_core_migration_type(), all_vaults=vaults)
        else:
            operate_vaults = vaults
        for v in operate_vaults:
            if is_migration_new(m, v):
                run_migration_at_vault(m, v, vaults)
                update_migration(m, v)
            new_version = get_vault_version(url=v.get('url'), token=v.get('token'))
            if new_version == m.get_version():
                msg = "vault [%s] upgraded successfully to version [%s]" % (v.get('name'), new_version)
                Console.info(msg)
            else:
                msg = "vault [%s] is NOT upgraded to version [%s]" % (v.get('name'), new_version)
                Console.info(msg)
                exit(1)


def list_migrations_command(args):
    migration_dir = args.migration_dir
    Console.info("Migrations in [%s]:" % migration_dir)
    Console.info("")
    migrations = load_migrations(migration_dir)

    for migration in migrations:
        version = migration.get_version()
        path = migration.path
        name = migration.name
        line = "%s\t%s\t%s" % (version, name, path)
        Console.info(line)


def main():
    parser = argparse.ArgumentParser()
    commands = parser.add_subparsers(help='commands')

    # add the create migration command
    create_cmd = commands.add_parser("create", help="create a new migration file")
    create_cmd.add_argument("name", help="the name of migration")
    create_cmd.add_argument("migration_dir", help="the migration directory")
    create_cmd.set_defaults(func=create_migration_command)

    # add the status command
    status_cmd = commands.add_parser("status", help="return status of vault and migrations")
    status_cmd.add_argument("migration_dir", help="the migration directory")
    status_cmd.set_defaults(func=print_status_command)

    args = parser.parse_args()
    try:
        func = args.func
    except AttributeError:
        parser.error("too few arguments")
    func(args)


if __name__ == '__main__':
    sys.exit(main())
