# negentropy-dev-terraform

Terraform definition of negentropy-dev project in GCP

```bash
export GOOGLE_PROJECT=negentropy-dev
terraform plan -out tf.plan
terraform apply "tf.plan"
```

one-time:
```bash
gcloud iam service-accounts keys create gar-github-runner.json --iam-account=gar-github-runner@negentropy-dev.iam.gserviceaccount.com
```

one-time, registrysecret:
```
kubectl create secret docker-registry registrysecret --docker-server=https://europe-west1-docker.pkg.dev --docker-email=gar-ro-user@negentropy-dev.iam.gserviceaccount.com --docker-username=_json_key_base64 --docker-password="$(cat gar-ro-user.json | base64 -w0)"
```