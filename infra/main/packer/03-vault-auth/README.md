### Certbot

Uses `dns-google` to obtain a certificate. 

Requires service account with the [following permissions](https://certbot-dns-google.readthedocs.io/en/stable/#credentials):

- `dns.changes.create`
- `dns.changes.get`
- `dns.managedZones.list`
- `dns.resourceRecordSets.create`
- `dns.resourceRecordSets.delete`
- `dns.resourceRecordSets.list`
- `dns.resourceRecordSets.update`

