#!/bin/sh
set -e

# Stop and disable service if it's running
if [ -d /run/systemd/system ]; then
    systemctl stop namedot >/dev/null 2>&1 || true
    systemctl disable namedot >/dev/null 2>&1 || true
fi

exit 0
