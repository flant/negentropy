import importlib
import os
import traceback

from config import UTC_LENGTH
from errors import InvalidMigrationError, InvalidNameError, Error


class Migration(object):
    """ This class represents a migration version. """

    def __init__(self, path):
        self.path = path  # ../../configurator/vault_migrations/conf/20210706132531_enable_pki/migrate.py
        self.filename = os.path.basename(os.path.dirname(path))  # 20210706132531_enable_pki
        self.dirname = os.path.basename(os.path.dirname(path))  # 20210706132531_enable_pki
        self.module_name, _ = os.path.splitext(self.filename)  # 20210706132531_enable_pki
        self.get_version()  # will assert the filename is valid
        self.name = self.module_name[UTC_LENGTH:]  # enable_pki
        
        # implement the update function from migrate script for using further
        while self.name.startswith('_'):
            self.name = self.name[1:]
        try:
            loader = importlib.machinery.SourceFileLoader(self.module_name, path)
            self.module = loader.load_module()
        except:
            raise InvalidMigrationError(f"Invalid migration {path}: {traceback.format_exc()}")
        
        # assert the migration has the needed methods
        if not self.has_method(self.module, 'upgrade'):
            raise InvalidMigrationError(f"Migration {self.path} is missing required methods: 'upgrade'.")

    def get_version(self):
        """the time stamp of this Migration"""
        if len(self.dirname) < UTC_LENGTH:
            raise InvalidNameError(self.filename)
        timestamp = self.dirname[:UTC_LENGTH]
        return timestamp

    def upgrade(self, conn):
        self.module.upgrade(conn)

    def get_core_migration_type(self) -> str:
        """return type of core migration, if applicable """
        parts = self.name.split("_")
        core_migration_type = parts[0]
        
        if core_migration_type not in ['auth', 'root', 'all']:
            raise Error('wrong type: %s of migration: %s' % (core_migration_type, self.name))
        return core_migration_type

    def __repr__(self):
        return f'Migration({self.filename})'
    
    def has_method(self, an_object, method_name):
        return hasattr(an_object, method_name) and \
           callable(getattr(an_object, method_name))
