#!/bin/sh
# Generate Mosquitto passwd from Docker secrets, then start Mosquitto.
# Secrets: api_password, bridge_password (mounted under /run/secrets/)

set -e
PASSWD_FILE="${PASSWD_FILE:-/mosquitto/data/passwd}"

if [ -f /run/secrets/api_password ] && [ -f /run/secrets/bridge_password ]; then
  api_pass="$(cat /run/secrets/api_password)"
  bridge_pass="$(cat /run/secrets/bridge_password)"
  mosquitto_passwd -b -c "$PASSWD_FILE" bridge "$bridge_pass"
  mosquitto_passwd -b "$PASSWD_FILE" api "$api_pass"
  chown mosquitto:mosquitto "$PASSWD_FILE"
else
  echo "Error: /run/secrets/api_password and /run/secrets/bridge_password must exist" >&2
  exit 1
fi

exec /usr/sbin/mosquitto -c /mosquitto/config/mosquitto.conf
