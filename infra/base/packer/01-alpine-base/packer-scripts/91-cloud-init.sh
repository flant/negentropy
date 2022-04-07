set -exu

cat <<'EOF' > /etc/cloud/cloud.cfg
users:
   - default

disable_root: false
preserve_hostname: true

cloud_init_modules:
 - migrator
 - seed_random
 - ssh
 - growpart
 - resizefs

system_info:
   distro: alpine
   default_user:
     name: root
     shell: /bin/bash
   paths:
      cloud_dir: /var/lib/cloud/
      templates_dir: /etc/cloud/templates/
   ssh_svcname: sshd
EOF

setup-cloud-init
