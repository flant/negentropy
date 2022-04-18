from typing import TypedDict, List

import hvac
import time


class Vault(TypedDict):
    name: str
    token: str
    url: str


# {name:policy}
policies = {'ssh': {
    'name': 'ssh',
    'roles': ['ssh'],
    'claim_schema': 'TODO',
    'allowed_auth_methods': ['multipass'],
    'rego': """
package negentropy


default requested_ttl = "600s"
default requested_max_ttl = "1200s"

requested_ttl = input.ttl
requested_max_ttl = input.max_ttl

filtered_bindings[r] {
#	tenant := input.tenant_uuid
    project := input.project_uuid
	some i
	r := data.effective_roles[i]
#   	data.effective_roles[i].tenant_uuid==tenant
    	data.effective_roles[i].projects[_]==project
        to_seconds_number(data.effective_roles[i].options.ttl)>=to_seconds_number(requested_ttl)
        to_seconds_number(data.effective_roles[i].options.max_ttl)>=to_seconds_number(requested_max_ttl)
}

default allow = false

allow {count(filtered_bindings) >0}

# пути по которым должен появится доступ
rules = [
	{"path":"ssh/sign/signer","capabilities":["update"]}
    ]{allow}

ttl := requested_ttl {allow}

max_ttl := requested_max_ttl {allow}

# Переводим в число секунд
to_seconds_number(t) = x {
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
	 x = value ; endswith(lower_t, "s")
}{
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
	 x = value*60 ; endswith(lower_t, "m")
}{
	 lower_t = lower(t)
     value = to_number(trim_right(lower_t, "hms"))
     x = value*3600 ; endswith(lower_t, "h")
}"""
}}

# need in case of first run to provide time for roles appears at auth plugins
time.sleep(5)


def upgrade(vault_name: str, vaults: List[Vault]):
    vault = next(v for v in vaults if v['name'] == vault_name)
    vault_client = hvac.Client(url=vault['url'], token=vault['token'])
    for name, policy in policies.items():
        print("INFO: create policy '{}' at '{}' vault".format(name, vault_name))
        vault_client.write(path='auth/flant_iam_auth/login_policy', **policy)
