# Development

For now it requires docker to run.

## Build and run

```bash
./dev-build.sh  # build plugins
./dev-start.sh  # start vault and kafka
̀```

### Single plugin

To build and start particular plugin, pass theit directory names to script arguments

```
./dev-build.sh flant_iam
./dev-start.sh flant_iam
```

### Access

IAM API is accessible on

Base URL:
- `http://localhost:8200/v1/flant_iam`

Headers:
- `Authorization: Bearer root`