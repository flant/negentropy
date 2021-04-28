set -eux

sed -i 's/^#HostKey/HostKey/g' /etc/ssh/sshd_config
sed -i 's#/etc/ssh/ssh_host#/tmp/etc/ssh/ssh_host#g' /etc/ssh/sshd_config
sed -i 's#ssh-keygen -A#mkdir -p /tmp/etc/ssh; ssh-keygen -A -f /tmp#g' /etc/init.d/sshd

# todo: uncomment ssh deletion
#apk del openssh
