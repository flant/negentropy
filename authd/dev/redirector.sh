#!/bin/bash

function patch_nginx_config() {
  CFG_PATH=/etc/nginx/conf.d/default.conf

  # redirect to other port
  sed -iorig \
     -e '/server_name\ \ localhost\;/s/\;/;\n    return 302 http:\/\/localhost:8200$request_uri;/' \
     $CFG_PATH

  cat $CFG_PATH
}

function run_nginx_container() {
  DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
  SCRIPT="$(basename "${BASH_SOURCE[0]}")"

  echo Run nginx accessible at http://localhost:8083/

  docker run --rm -ti \
    -v $DIR/$SCRIPT:/docker-entrypoint.d/00-$SCRIPT \
    --name vault.redirector \
    -p 8083:80 \
    nginx:latest
}

if [ -f /.dockerenv ]; then
  patch_nginx_config
else
  run_nginx_container
fi