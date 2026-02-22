# Mosquitto MQTT Broker Configuration

## Broker Layout

The project uses two Mosquitto broker roles:

- **Local broker** (`local-broker/`) – Runs near the gateway. The gateway publishes to this broker; it bridges selected topics to the remote broker.
- **Remote broker** (`remote-broker/`) – Central/hosted broker. Local brokers bridge to it; the API subscribes here.

Configuration files (`mosquitto.conf`, `mosquitto.acl`) are already in `local-broker/` and `remote-broker/`. You only need to create or update the password files.

## Setting Up Authentication

### Remote broker (`remote-broker/passwd`)

Users: `bridge` (used by local broker bridge), `api` (used by the API to subscribe to telemetry).

```bash
# Create password file with bridge user (required for local→remote bridge)
mosquitto_passwd -c -b remote-broker/passwd bridge "your-bridge-password"

# Add api user (for API subscribing to telemetry)
mosquitto_passwd -b remote-broker/passwd api "your-api-password"
```

### Local broker (`local-broker/passwd`)

User: `gateway` (used by gateway devices to publish telemetry).

```bash
mosquitto_passwd -c -b local-broker/passwd gateway "your-gateway-password"
```

### Updating passwords

```bash
# Remote broker
mosquitto_passwd -b remote-broker/passwd bridge "new-bridge-password"
mosquitto_passwd -b remote-broker/passwd api "new-api-password"

# Local broker
mosquitto_passwd -b local-broker/passwd gateway "new-gateway-password"
```

## Environment Variables

For the **API** (`api/.env`), set at least:

```env
MQTT_BROKER=localhost
MQTT_PORT=1885
MQTT_USERNAME=api
MQTT_PASSWORD=your-api-password
MQTT_TOPIC=stations/+/telemetry
```

For dev, the remote broker is exposed on host port 1885 (see `docker-compose.dev.yaml`). The password must match the one used in `remote-broker/passwd` for user `api` and `bridge`. After changing the passwd file.
Local broker is exposed on host port 1884 (see `docker-compose.dev.yaml`). The password must match the one used in `local-broker/passwd` for user `gateway`. After changing the passwd file. Restart the broker container so it reloads.