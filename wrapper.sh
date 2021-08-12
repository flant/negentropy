#!/usr/bin/env bash

nss_databases=(passwd group shadow)

for i in "${nss_databases[@]}"
do
  check_if_exist=$(cat /etc/nsswitch.conf | grep -w $i | grep -oE '[^ ]+$')
  if [[ $check_if_exist == flantauth ]]; then
    echo "$i: already configured"
  else
    sed -i '/\<'"$i"'\>/s/$/ flantauth/' /etc/nsswitch.conf
  fi
done

mkdir /run/sshd

/usr/sbin/sshd -d
