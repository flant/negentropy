set -exu

apk add logrotate

mv /etc/periodic/daily/logrotate /etc/periodic/15min/

cat <<'EOF' > /etc/logrotate.conf
weekly

# Keep 4 rotations worth of backlogs (size x 4).
rotate 4

# Max size of log file (rotate log file when this size is reached).
size 1M

# Truncate the original log file in place after creating a copy.
copytruncate

# If the log file is missing, go on to the next one without issuing an error message.
missingok

# Old versions of log files are compressed with gzip.
compress

# We truncating existed files therefore file owners won't be changed.
su root root

# Rotate all logs in the first level of /var/log.
/var/log/* {}

EOF
