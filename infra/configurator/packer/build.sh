#!/usr/bin/env bash

for d in */ ; do
  echo "Building image from directory $d"
  output=$(cd "$d"; ./build.sh)
  code=$?
  echo "$output"
  if [ $code -eq 0 ]; then
    echo "Succeed."
  else
    echo "Error occurred."
    exit 1
  fi
done
