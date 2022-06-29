set -exu

# Disable Core Dumps
cat <<'EOF' > /etc/sysctl.d/local.conf
kernel.core_pattern=/dev/null
EOF

# Make tmpfs
rc-service syslog stop
echo "tmpfs /tmp tmpfs defaults,size=200M 0 0" >> /etc/fstab
echo "tmpfs /var/log tmpfs defaults,size=200M 0 0" >> /etc/fstab
echo "tmpfs /var/lib tmpfs defaults,size=10240M 0 0" >> /etc/fstab
rm -rf /var/log && mkdir /var/log && mount /var/log
rm -rf /var/lib && mkdir /var/lib && mount /var/lib
rc-service syslog start

# Create a symlink to /tmp for /etc/resolv.conf
touch /tmp/dhcpcd.resolv.conf
rm -f /etc/resolv.conf
ln -s /tmp/dhcpcd.resolv.conf /etc/resolv.conf
# Change `resolv.conf` path in the udhcpc script
sed -i 's#/etc/resolv.conf#/tmp/dhcpcd.resolv.conf#g' /usr/share/udhcpc/default.script

# Create a symlink to /tmp for /etc/hostname
cp /etc/hostname /tmp/hostname
rm -f /etc/hostname
ln -s /tmp/hostname /etc/hostname

# Create handler for setting hostname
cat <<'EOF' > /bin/update-hostname
#!/usr/bin/env bash
hostname="$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/hostname" -H "Metadata-Flavor: Google" 2>/dev/null)"
host=${hostname%%.*}
ip=$(ip r get 1 | awk '{print $7}')
echo "$host" > /tmp/hostname
hostname "$host"
if grep -Fq "${ip}" /etc/hosts
then
    echo "/etc/hosts already configured"
else
    echo "$ip	$hostname $host" >> /etc/hosts
fi
EOF

chmod +x /bin/update-hostname

# Set bash as default shell
sed -i 's#/bin/ash#/bin/bash#g' /etc/passwd
