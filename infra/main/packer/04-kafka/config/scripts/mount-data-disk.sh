#!/usr/bin/env bash

if [[ "$(fdisk -l /dev/sdb 2>/dev/null | grep sdb1 | wc -l)" != "1" ]]; then
	(
	echo n # Add a new partition
	echo p # Primary partition
	echo 1 # Partition number
	echo   # First sector (Accept default: 1)
	echo   # Last sector (Accept default: varies)
	echo w # Write changes
	) | fdisk /dev/sdb
	mkfs.ext4 /dev/sdb1
fi

if [[ "$(mount | grep sdb1 | wc -l)" != "1" ]]; then
  mount /dev/sdb1 /data
fi

chown kafka:kafka /data
