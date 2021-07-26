set -exu

sed -i 's#/var/lib tmpfs defaults,size=1000M#/var/lib tmpfs defaults,size=10000M#' /etc/fstab
