import datetime
import glob
import importlib.machinery
import os
import traceback
import hvac
import sys
import argparse


#TODO
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
        self.path = path
        self.filename = os.path.basename(os.path.dirname(path))
        self.dirname = os.path.basename(os.path.dirname(path))
        self.module_name, _ = os.path.splitext(self.filename)
        self.get_version() # will assert the filename is valid
        self.name = self.module_name[UTC_LENGTH:]
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

    def __repr__(self):
        return 'Migration(%s)' % self.filename

class Vault(object):

    def __init__(self):
        self.conn = hvac.Client(
            url=os.environ['VAULT_ADDR'],
            token=os.environ['VAULT_TOKEN'],
        )

    def is_version_controlled(self):
        try:
            ret = self.conn.secrets.kv.read_secret_version(path=VERSION_KEY)
        except:
            ret = False
        return bool(ret)

    def upgrade(self, migrations, target_version=None):
        if target_version:
            _assert_migration_exists(migrations, target_version)

        migrations.sort(key=lambda x: x.get_version())
        vault_version = self.get_version()
        for migration in migrations:
            current_version = migration.get_version()
            if current_version <= vault_version:
                continue
            if target_version and current_version > target_version:
                break
            Console.info("")
            Console.info(">>>>>> starting migration %s" % migration.get_version())
            migration.upgrade(self.conn)
            new_version = migration.get_version()
            self.update_version(new_version)

    def get_version(self):
        """ Return the vault's version, or None if it is not under version
            control.
        """
        if not self.is_version_controlled():
            self.initialize_version_control()
        result = self.conn.secrets.kv.read_secret_version(path=VERSION_KEY)
        return result['data']['data']['version'] if result else "0"

    def update_version(self, version):
        Console.info("====== vault updated to version %s" % version)
        self.conn.secrets.kv.create_or_update_secret(path=VERSION_KEY,secret=dict(version=version))

    def initialize_version_control(self):
        self.conn.secrets.kv.create_or_update_secret(path=VERSION_KEY,secret=dict(version='0'))

    def __repr__(self):
        return 'Vault()' 

def _assert_migration_exists(migrations, version):
    if version not in (m.get_version() for m in migrations):
        raise Error('No migration with version %s exists.' % version)

def load_migrations(directory):
    """ Return the migrations contained in the given directory. """
    if not is_directory(directory):
        msg = "%s is not a directory." % directory
        raise Error(msg)
    wildcard = os.path.join(directory, '*' ,'migrate.py')
    migration_files = glob.glob(wildcard)
    return [Migration(f) for f in migration_files]
   
def upgrade(migration_dir, version=None):
    """ Upgrade the given vault with the migrations contained in the
        migrations directory. If a version is not specified, upgrade
        to the most recent version.
    """
    db = Vault()
    if not db.is_version_controlled():
        db.initialize_version_control()
    migrations = load_migrations(migration_dir)
    db.upgrade(migrations, version)

def get_latest_migration_version(migration_dir):
    migrations = load_migrations(migration_dir)
    migrations.sort(key=lambda x: x.get_version())
    latest_migration = migrations[-1:]
    if latest_migration:
      return latest_migration[0].get_version()
    return None

def get_vault_version():
    """ Return the migration version of the given vault. """
    db = Vault()
    return db.get_version()

def create_migration(name, directory=None):
    """ Create a migration with the given name. If no directory is specified,
        the current working directory will be used.
    """
    directory = directory if directory else '.'
    now = datetime.datetime.now()
    version = now.strftime("%Y%m%d%H%M%S")

    contents = MIGRATION_TEMPLATE % {'name':name, 'version':version}

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


def upgrade_db_command(args):
    migration_dir = args.migration_dir
    version = args.version

    current_vault_version = get_vault_version()
    msg = 'vault version            [%s]' % (current_vault_version)
    Console.info(msg)

    latest_migration_version = get_latest_migration_version(migration_dir)
    msg = 'latest migration version [%s]' % (latest_migration_version)
    Console.info(msg)

    if latest_migration_version == current_vault_version:
      Console.info('Vault is already up-to-date! Nothing to do.') 
      return 

    msg = 'upgrading vault to most recent version...'
    if version:
        msg = 'upgrading vault to version [%s]' % (version)
    Console.info(msg)
    
    upgrade(migration_dir, version)
    new_version = get_vault_version()
    
    if version:
        assert new_version == version
    
    msg = "upgraded successfully to version [%s]" % (new_version)
    Console.info(msg)

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


    # add the upgrade command
    migrate_cmd = commands.add_parser("migrate", help="upgrade the vault. If a version isn't specified, upgrade to the most recent version.")
    migrate_cmd.add_argument("migration_dir", help="the migration directory")
    migrate_cmd.add_argument("-v", "--version", help="the target migration version")
    migrate_cmd.set_defaults(version=None)
    migrate_cmd.set_defaults(func=upgrade_db_command)

    args = parser.parse_args()
    try:
        func = args.func
    except AttributeError:
        parser.error("too few arguments")
    func(args)

if __name__ == '__main__':
    sys.exit(main())