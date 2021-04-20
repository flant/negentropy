set -exu

# Disable Core Dumps
cat <<'EOF' > /etc/sysctl.d/local.conf
kernel.core_pattern=/dev/null
EOF

# Make tmpfs
rc-service syslog stop
echo "tmpfs /tmp tmpfs defaults,size=100M 0 0" >> /etc/fstab
echo "tmpfs /var/log tmpfs defaults,size=200M 0 0" >> /etc/fstab
echo "tmpfs /var/lib tmpfs defaults,size=5M 0 0" >> /etc/fstab
rm -rf /var/log && mkdir /var/log && mount /var/log
rm -rf /var/lib && mkdir /var/lib && mount /var/lib
rc-service syslog start

# Create a symlink to /tmp for /etc/resolv.conf
touch /tmp/dhcpcd.resolv.conf
rm -rf /etc/resolv.conf
ln -s /tmp/dhcpcd.resolv.conf /etc/resolv.conf
# Change `resolv.conf` path in the udhcpc script
sed -i 's#/etc/resolv.conf#/tmp/dhcpcd.resolv.conf#g' /usr/share/udhcpc/default.script
