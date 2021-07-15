#!/usr/bin/env bash

. /etc/kafka/scripts/variables.sh

# Ensure the certificate renewal job is in the crontab.
if ! grep -q "renew-certificate.sh" /etc/crontabs/root; then
  # To prevent synchronous certificate renewal we drift job startup time according to node id.
  minute=0
  for ((i=1; i<=$NODE_ID; i++)); do
      minute=$(($minute+10))
      if [ $minute -ge 60 ]; then
        minute=0
      fi
  done
  echo "$minute	2	*	*	*	/etc/kafka/scripts/renew-certificate.sh" >> /etc/crontabs/root
  rc-service crond restart
fi
