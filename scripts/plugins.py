from consts import FLANT_IAM, FLANT_IAM_AUTH, ROOT_FLANT_IAM_SELF_TOPIC, negentropy_plugins
from vault import Vault, check_response

PublicKey = str
VaultName = str
PluginName = str


class Plugin:
    """ The Plugin class represents plugin of negentropy"""

    def __init__(
            self,
            vault_name: VaultName,
            name: PluginName,
            plugin_public_key: PublicKey,
    ):
        self.vault_name = vault_name
        self.name = name
        self.plugin_public_key = plugin_public_key
        self.self_topic_name = None  # not configured yet
        self.root_topic_name = None  # not configured yet


def collect_all_not_flant_iam_keys(plugins: list[Plugin]) -> list[PublicKey]:
    """ collect_all_not_flant_iam_keys collects all not flant_iam keys """
    result = []
    for plugin in plugins:
        if plugin.name is not FLANT_IAM:
            result.append(plugin.plugin_public_key)
    return result


def vaults_by_name(vaults: list[Vault]) -> dict[VaultName, Vault]:
    """ map list to dict """
    return {vault.name: vault for vault in vaults}


def configure_pubkeys_and_replicas_at_master_root_vault(master_root_vault: Vault,
                                                        plugins: list[Plugin]) -> list[Plugin]:
    """ configure pubkeys and replicas, returns updated plugins"""
    flant_iam_kafka_configure(master_root_vault, collect_all_not_flant_iam_keys(plugins))
    return flant_iam_replicas_configure(master_root_vault, plugins)


def connect_kafka_and_generate_key(vault: Vault, plugin: PluginName, kafka_endpoints: str) -> PublicKey:
    """returns public key of plugin
       Example:
    -----BEGIN RSA PUBLIC KEY-----
    MIIBCgKCAQEA61BjmfXGEvWmegnBGSuS+rU9soUg2FnODva32D1AqhwdziwHINFa
    ...
    -----END RSA PUBLIC KEY-----
    """
    # generate csr
    check_response(vault.write_to_plugin(plugin, "kafka/generate_csr?force=true"), 200)
    # connect kafka
    check_response(
        vault.write_to_plugin(plugin, "kafka/configure_access", {"kafka_endpoints": kafka_endpoints}), 204)
    # get public_key
    resp = check_response(vault.read_from_plugin(plugin, "kafka/public_key"), 200)
    pk = resp.json().get("data").get("public_key")
    if pk is None:
        raise Exception("need 'data.public_key' in response:{}".format(resp.text))
    return pk


def find_master_root_vault(vaults: list[Vault]) -> Vault:
    """ returns master root vault (now it is just first vault with flant_iam onboard) """
    for vault in vaults:
        if FLANT_IAM in vault.plugin_names:
            return vault
    raise Exception("there is no vaults with flant_iam onboard in passed: {}".format(vaults))


def flant_iam_kafka_configure(vault: Vault, peers_pub_keys: list[PublicKey]):
    """flant_iam_kafka_configure configures flant_iam, getting public keys of all others plugins of all others vaults"""
    check_response(
        vault.write_to_plugin(plugin=FLANT_IAM, path="kafka/configure",
                              json={
                                  "self_topic_name": ROOT_FLANT_IAM_SELF_TOPIC,
                                  "peers_public_keys": peers_pub_keys
                              }))


ReplicaName = str
ReplicaType = str
PluginPublicKey = PublicKey


def synonym_name_by_plugin_name(plugin_name: str) -> str:
    return "auth" if plugin_name == FLANT_IAM_AUTH else plugin_name


def flant_iam_replicas_configure(master_vault: Vault, plugins: list[Plugin]) -> list[Plugin]:
    """ flant_iam_replicas_configure provide configuring flant_iam & kafka topics for data replication """
    result = []
    plugin_counter = dict()
    for plugin in plugins:
        if plugin.name == FLANT_IAM:
            plugin.self_topic_name = ROOT_FLANT_IAM_SELF_TOPIC
            plugin.root_topic_name = "not applicable"
            result.append(plugin)
        else:
            idx = plugin_counter.get(plugin.name) + 1 if plugin_counter.get(plugin.name) else 1
            plugin_counter[plugin.name] = idx
            synonym = synonym_name_by_plugin_name(plugin.name)
            replica_name = synonym + "-" + str(idx)  # auth-1
            plugin.self_topic_name = synonym + "-source." + replica_name  # auth-source.auth-1
            plugin.root_topic_name = ROOT_FLANT_IAM_SELF_TOPIC + "." + replica_name  # root_source.auth-1
            check_response(
                master_vault.write_to_plugin(plugin=FLANT_IAM, path="replica/" + replica_name,
                                             # flant_iam/replica/auth-1
                                             json={
                                                 "type": "Vault",
                                                 "public_key": plugin.plugin_public_key
                                             }))
            result.append(plugin)
    return result


def plugin_kafka_configure(vault: Vault, plugin: Plugin, root_flant_iam_public_key: PublicKey):
    check_response(
        vault.write_to_plugin(plugin=plugin.name, path="kafka/configure", json={
            "peers_public_keys": root_flant_iam_public_key,
            "self_topic_name": plugin.self_topic_name,  # "auth-source.auth-1"
            "root_topic_name": plugin.root_topic_name,  # "root_source.auth-1",
            "root_public_key": root_flant_iam_public_key,
        }))


def find_key_by_plugin_and_vault_name(plugins: list[Plugin], root_vault_name: VaultName) -> PublicKey:
    for plugin in plugins:
        if plugin.vault_name == root_vault_name and plugin.name == FLANT_IAM:
            return plugin.plugin_public_key
    raise Exception(
        "there is no plugin with name '{}' at vault '{}' in passed plugins:{}".format(FLANT_IAM, root_vault_name,
                                                                                      plugins))


def connect_plugins(vaults: list[Vault], kafka_endpoints: str) -> list[Plugin]:
    """ connect negentropy plugins to kafka """
    """ THE MAIN FUNCTION """
    plugins = []
    print("connect_plugins started, configure kafka endpoints:")
    for vault in vaults:
        print("\t vault:{}".format(vault.name))
        for plugin_name in vault.plugin_names:
            if plugin_name in negentropy_plugins:
                pk = connect_kafka_and_generate_key(vault, plugin_name, kafka_endpoints)
                plugins.append(Plugin(vault_name=vault.name, name=plugin_name, plugin_public_key=pk))
                print("\t\tplugin:{}".format(plugin_name))

    master_root_vault = find_master_root_vault(vaults)
    print("master root vault is '{}' at '{}".format(master_root_vault.name, master_root_vault.url))
    root_flant_iam_public_key = find_key_by_plugin_and_vault_name(plugins, master_root_vault.name)
    print("start configure_pubkeys_and_replicas_at_master_root_vault for plugins:{}".
          format([p.name + " at " + p.vault_name for p in plugins]))
    plugins = configure_pubkeys_and_replicas_at_master_root_vault(master_root_vault, plugins)
    print("configure_pubkeys_and_replicas_at_master_root_vault is done")
    vaults_dict = vaults_by_name(vaults)
    print("start plugin_kafka_configure for plugins:")
    for plugin in plugins:
        print("\t vault:{}".format(vault.name))
        if plugin.name != FLANT_IAM:
            print("\t\tplugin:{}".format(plugin.name))
            vault = vaults_dict[plugin.vault_name]
            plugin_kafka_configure(vault, plugin, root_flant_iam_public_key)
    print("plugin_kafka_configure is done")
    return plugins
