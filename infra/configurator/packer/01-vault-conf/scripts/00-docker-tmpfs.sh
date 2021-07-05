set -exu

# Docker in this image needs larger /var/lib tmpfs
sed -i 's#/var/lib tmpfs defaults,size=1000M#/var/lib tmpfs defaults,size=10000M#' /etc/fstab
