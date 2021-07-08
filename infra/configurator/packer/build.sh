#!/usr/bin/env bash

for d in */ ; do
  echo "Building image from directory $d"
  output=$(cd "$d"; ./build.sh)
  if [ $? -eq 0 ]; then
    echo "Succeed: "
  else
    echo "Error occurred: "
  fi
  echo "$output"
done
