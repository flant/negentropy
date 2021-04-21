# flant_iam

The IAM module that manages tenants, projects, users, and roles.

## Development

### Start vault

To build the plugin, start vault in docker and register the plugin: `./start.sh`. This script also populates test data.

### Run tests

To run API tests, run `./test.sh` or `yarn test` from the tests directory. Tests rely on setup from `./start.sh` and the
data it genrated and put into tests/data directory.
