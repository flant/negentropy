# flant_iam

The IAM module that manages tenants, projects, users, and roles.

## Development

### Start vault

To build the plugin, start vault in docker and register the 
plugin: `./start.sh`. This script also populates test data.

### Run tests

After `start.sh` is run, do

```shell
make deps 
make test
```
