#!/usr/bin/env bash

for d in */ ; do
  if [ "$d" == "01-alpine-base/" ]; then
    continue
  fi
  echo "Building image from directory $d"
  output=$(cd "$d"; ./build.sh)
  if [ $? -eq 0 ]; then
    echo "Succeed"
  else
    echo "Error occurred:"
    echo "$output"
  fi
done
