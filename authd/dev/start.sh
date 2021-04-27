#!/bin/bash

dir=$(pwd)

authd="$dir/authd"
conf_dir="$dir/dev/conf"
if [[ -e $dir/start.sh ]]; then
  cd ..
fi

$authd --conf-dir=$conf_dir
