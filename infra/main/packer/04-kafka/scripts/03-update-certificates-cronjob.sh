set -exu

echo '0	2	*	*	*    /etc/kafka/scripts/update-certificates-cronjob.sh' >> /etc/crontabs/root
