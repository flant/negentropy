set -eux

grep -Eq ".*PasswordAuthentication (yes|no)" /etc/ssh/sshd_config && sed -i -e "s/.*PasswordAuthentication \(yes\|no\)/PasswordAuthentication no/g" /etc/ssh/sshd_config || echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
passwd -d root
