#!/bin/bash

# run this on the victim to uninstall the backdoor

# 1. Kill the running process
systemctl stop sys_update.service

# 2. Remove it from the boot sequence
systemctl disable sys_update.service

# 3. Delete your payload and the service file
rm /usr/local/bin/.sys_update
rm /etc/systemd/system/sys_update.service

# 4. Tell the OS to refresh its service list so it stops looking for it
systemctl daemon-reload
