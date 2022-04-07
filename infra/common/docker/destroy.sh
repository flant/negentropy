#!/usr/bin/env bash

if [ "$1" != "configurator" ] && [ "$1" != "main" ]; then
  echo "invalid argument"
  exit 1
fi

for d in $(ls ../../$1/terraform/ | sort -r); do
  echo "destroy terraform state in $d"
  terraform -chdir=../../$1/terraform/$d destroy -auto-approve
done



