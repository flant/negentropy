#!/bin/bash

set -e

[ -z "$NS" ] && echo "NS env variable required" && exit 1
[ -z "$GIT_BRANCH" ] && echo "GIT_BRANCH env variable required" && exit 1
[ -z "$GIT_REPO" ] && GIT_REPO=https://github.com/flant/negentropy.git
[ -z "$REQUIRED_NUMBER_OF_SIGNATURES" ] && REQUIRED_NUMBER_OF_SIGNATURES=0
[ -z "$INITIAL_LAST_SUCCESSFULL_COMMIT" ] && INITIAL_LAST_SUCCESSFULL_COMMIT=""
[ -z "$GIT_POLL_PERIOD" ] && GIT_POLL_PERIOD=1m

echo "================= CONFIG ================"
echo "NAMESPACE: $NS"
echo "GIT_REPO: $GIT_REPO"
echo "GIT_BRANCH: $GIT_BRANCH"
echo "REQUIRED_NUMBER_OF_SIGNATURES: $REQUIRED_NUMBER_OF_SIGNATURES"
echo "INITIAL_LAST_SUCCESSFULL_COMMIT: $INITIAL_LAST_SUCCESSFULL_COMMIT"
echo "GIT_POLL_PERIOD: $GIT_POLL_PERIOD"
echo
echo "===== BEGIN NEGENTROPY BOOTSTRAPING ====="
echo
set -x

kubectl create ns $NS

kubectl -n $NS create configmap bootstrap --from-literal=GIT_REPO=$GIT_REPO --from-literal=GIT_BRANCH=$GIT_BRANCH --from-literal=REQUIRED_NUMBER_OF_SIGNATURES=$REQUIRED_NUMBER_OF_SIGNATURES --from-literal=GIT_POLL_PERIOD=$GIT_POLL_PERIOD --from-literal=INITIAL_LAST_SUCCESSFULL_COMMIT="$INITIAL_LAST_SUCCESSFULL_COMMIT"

kubectl -n $NS create sa deploy
kubectl -n $NS label sa deploy "app.kubernetes.io/managed-by"=Helm
kubectl -n $NS annotate sa deploy "meta.helm.sh/release-name"=negentropy
kubectl -n $NS annotate sa deploy "meta.helm.sh/release-namespace"=$NS

kubectl -n $NS create role deploy --verb='*' --resource='*.*'
kubectl -n $NS label role deploy "app.kubernetes.io/managed-by"=Helm
kubectl -n $NS annotate role deploy "meta.helm.sh/release-name"=negentropy
kubectl -n $NS annotate role deploy "meta.helm.sh/release-namespace"=$NS

kubectl -n $NS create rolebinding deploy --role=deploy --serviceaccount=$NS:deploy
kubectl -n $NS label rolebinding deploy "app.kubernetes.io/managed-by"=Helm
kubectl -n $NS annotate rolebinding deploy "meta.helm.sh/release-name"=negentropy
kubectl -n $NS annotate rolebinding deploy "meta.helm.sh/release-namespace"=$NS


(echo -e "metadata:\n  name: bootstrap" ; curl -s https://raw.githubusercontent.com/flant/negentropy/$GIT_BRANCH/vault-plugins/flant_gitops/pkg/kube/job_template.yaml) | kubectl -n $NS create -f -
set +x

echo "========= BOOTSTRAP JOB RUNNING ========="
echo "To track deploy type: kubectl -n $NS logs job.batch/bootstrap -f"