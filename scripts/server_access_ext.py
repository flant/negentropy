from vault import Vault, check_response
from consts import FLANT_IAM, negentropy_plugins


class ServerAccessExtension:
    def __init__(
            self,
            vaults: list[Vault],
            roles_for_servers: list[str] = ["servers"],
            role_for_ssh_access: str = "ssh",
            delete_expired_password_seeds_after: int = 1000000,
            expire_password_seed_after_reveal_in: int = 1000000,
            last_allocated_uid: int = 10000,
    ):
        if vaults is None:
            raise Exception("definitely needs vaults for extension")
        self.vaults = vaults
        self.roles_for_servers = roles_for_servers
        self.role_for_ssh_access = role_for_ssh_access
        self.delete_expired_password_seeds_after = delete_expired_password_seeds_after
        self.expire_password_seed_after_reveal_in = expire_password_seed_after_reveal_in
        self.last_allocated_uid = last_allocated_uid
        """Creates a new ServerAccessExtension instance for configuring plugins.
        :param vault: vault instance for configuring extension
        :type vault: Vault        
        :param roles_for_servers: list of roles, which are suitable for using in server_access at servers  
        :type roles_for_servers: list[str]
        :param role_for_ssh_access: role for user(service_account) to get access to servers
        :type role_for_ssh_access: str
        :param delete_expired_password_seeds_after: 
        :type delete_expired_password_seeds_after: int
        :param expire_password_seed_after_reveal_in:
        :type expire_password_seed_after_reveal_in: int
        :param last_allocated_uid: 
        :type last_allocated_uid: int
        """

    def configure_extension_at_vaults(self):
        print ("ServerAccess: configure_extension_at_vaults:")
        for vault in self.vaults:
            print("\tvault:'{}'".format(vault.name))
            for plugin_name in vault.plugin_names:
                if plugin_name in negentropy_plugins:
                    print("\t\tplugin:'{}'".format(plugin_name))
                    if plugin_name == FLANT_IAM:
                        body = {
                            "roles_for_servers": self.roles_for_servers,
                            "role_for_ssh_access": self.role_for_ssh_access,
                            "delete_expired_password_seeds_after": self.delete_expired_password_seeds_after,
                            "expire_password_seed_after_reveal_in": self.expire_password_seed_after_reveal_in,
                            "last_allocated_uid": self.last_allocated_uid
                        }
                    else:
                        body = {
                            "role_for_ssh_access": self.role_for_ssh_access
                        }
                    check_response(
                        vault.write_to_plugin(plugin=plugin_name, path="configure_extension/server_access", json=body))
