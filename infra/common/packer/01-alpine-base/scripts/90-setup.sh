set -exu

# Enable community repository
sed -e 's;^#http\(.*\)/v3.13/community;http\1/v3.13/community;g' -i /etc/apk/repositories

# Disable Core Dumps
cat <<'EOF' > /etc/sysctl.d/local.conf
kernel.core_pattern=/dev/null
EOF

# Make tmpfs
rc-service syslog stop
echo "tmpfs /tmp tmpfs defaults,size=100M 0 0" >> /etc/fstab
echo "tmpfs /var/log tmpfs defaults,size=200M 0 0" >> /etc/fstab
echo "tmpfs /var/lib tmpfs defaults,size=1000M 0 0" >> /etc/fstab
echo "tmpfs /opt tmpfs defaults,size=100M 0 0" >> /etc/fstab
rm -rf /var/log && mkdir /var/log && mount /var/log
rm -rf /var/lib && mkdir /var/lib && mount /var/lib
rm -rf /opt && mkdir /opt && mount /opt
rc-service syslog start

# Create a symlink to /tmp for /etc/resolv.conf
touch /tmp/dhcpcd.resolv.conf
rm -rf /etc/resolv.conf
ln -s /tmp/dhcpcd.resolv.conf /etc/resolv.conf
# Change `resolv.conf` path in the udhcpc script
sed -i 's#/etc/resolv.conf#/tmp/dhcpcd.resolv.conf#g' /usr/share/udhcpc/default.script

# Create a symlink to /tmp for /etc/hostname
cp /etc/hostname /tmp/hostname
rm -rf /etc/hostname
ln -s /tmp/hostname /etc/hostname

# Create handler for setting hostname
cat <<'EOF' > /bin/update-hostname
#!/usr/bin/env bash
hostname="$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/hostname" -H "Metadata-Flavor: Google" 2>/dev/null)"
host=${hostname%%.*}
echo "$host" > /tmp/hostname
hostname "$host"
exit 0
EOF

chmod +x /bin/update-hostname
