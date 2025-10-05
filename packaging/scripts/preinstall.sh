#!/bin/sh
set -e

# Create namedot user and group if they don't exist
if ! getent group namedot >/dev/null; then
    groupadd -r namedot
fi

if ! getent passwd namedot >/dev/null; then
    useradd -r -g namedot -d /var/lib/namedot -s /sbin/nologin \
        -c "namedot DNS server" namedot
fi

exit 0
