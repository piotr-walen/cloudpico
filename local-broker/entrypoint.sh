#!/bin/sh
# Generate Mosquitto passwd from Docker secret, then start Mosquitto.
# Secret: gateway_password (mounted under /run/secrets/)

set -e
PASSWD_FILE="${PASSWD_FILE:-/mosquitto/data/passwd}"

if [ -f /run/secrets/gateway_password ]; then
  gateway_pass="$(cat /run/secrets/gateway_password)"
  mosquitto_passwd -b -c "$PASSWD_FILE" gateway "$gateway_pass"
  chown mosquitto:mosquitto "$PASSWD_FILE"
else
  echo "Error: /run/secrets/gateway_password must exist" >&2
  exit 1
fi

exec /usr/sbin/mosquitto -c /mosquitto/config/mosquitto.conf
