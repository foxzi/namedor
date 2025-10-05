#!/bin/sh
set -e

# Set proper ownership
chown -R namedot:namedot /var/lib/namedot /var/log/namedot

# Reload systemd
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
fi

# Print installation message
cat <<EOF

namedot has been installed successfully!

Configuration file: /etc/namedot/config.yaml
Data directory: /var/lib/namedot
Logs directory: /var/log/namedot

IMPORTANT: Edit /etc/namedot/config.yaml and change the default API token!

To start the service:
  sudo systemctl start namedot
  sudo systemctl enable namedot

To check status:
  sudo systemctl status namedot

EOF

exit 0
