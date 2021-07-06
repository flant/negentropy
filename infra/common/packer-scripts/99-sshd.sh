set -eux

sed -i 's/^#HostKey/HostKey/g' /etc/ssh/sshd_config
sed -i 's#/etc/ssh/ssh_host#/tmp/etc/ssh/ssh_host#g' /etc/ssh/sshd_config
sed -i 's#ssh-keygen -A#mkdir -p /tmp/etc/ssh; ssh-keygen -A -f /tmp#g' /etc/init.d/sshd

# todo: uncomment ssh deletion
#apk del openssh

mkdir -p /root/.ssh
echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTXjTmx3hq2EPDQHWSJN7By1VNFZ8colI5tEeZDBVYAe9Oxq4FZsKCb1aGIskDaiAHTxrbd2efoJTcPQLBSBM79dcELtqfKj9dtjy4S1W0mydvWb2oWLnvOaZX/H6pqjz8jrJAKXwXj2pWCOzXerwk9oSI4fCE7VbqsfT4bBfv27FN4/Vqa6iWiCc71oJopL9DldtuIYDVUgOZOa+t2J4hPCCSqEJK/r+ToHQbOWxbC5/OAufXDw2W1vkVeaZUur5xwwAxIb3wM3WoS3BbwNlDYg9UB2D8+EZgNz1CCCpSy1ELIn7q8RnrTp0+H8V9LoWHSgh3VCWeW8C/MnTW90IR tfadm' > /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
