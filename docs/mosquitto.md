# Mosquitto MQTT Broker Configuration

## Broker Layout

The project uses two Mosquitto broker roles:

- **Local broker** (`local-broker/`) – Runs near the gateway. The gateway publishes to this broker; it bridges selected topics to the hosted broker.
- **Hosted broker** (`hosted-broker/`) – Central broker. Local brokers bridge to it; the server subscribes here.

Configuration files (`mosquitto.conf`, `mosquitto.acl`) are already in `local-broker/` and `hosted-broker/`. You only need to create the password files.

## Setting Up Authentication

### Hosted broker (`hosted-broker/passwd`)

Users: `bridge` (used by local broker bridge), `server` (used by the app server).

```bash
# Create password file with bridge user (required for local→hosted bridge)
mosquitto_passwd -c -b hosted-broker/passwd bridge "your-bridge-password"

# Add server user (for server subscribing to telemetry)
mosquitto_passwd -b hosted-broker/passwd server "your-server-password"
```

### Local broker (`local-broker/passwd`)

User: `gateway` (used by gateway devices to publish telemetry).

```bash
mosquitto_passwd -c -b local-broker/passwd gateway "your-gateway-password"
```

### Updating passwords

```bash
# Hosted broker
mosquitto_passwd -b hosted-broker/passwd bridge "new-bridge-password"
mosquitto_passwd -b hosted-broker/passwd server "new-server-password"

# Local broker
mosquitto_passwd -b local-broker/passwd gateway "new-gateway-password"
```

## Environment Variables

Set these for docker-compose (or in a `.env` file in the project root):

```bash
export MQTT_API_PASSWORD="your-server-password"    # server user on hosted broker
export MQTT_GATEWAY_PASSWORD="your-gateway-password"  # gateway user on local broker
export MQTT_BRIDGE_PASSWORD="your-bridge-password"    # bridge user on hosted broker (used by local broker)
export MQTT_HOSTED_BROKER_ADDRESS="host:1883"         # only for local broker (e.g. mosquitto-hosted:1883 in dev)
```

Or in `.env`:

```env
MQTT_API_PASSWORD=your-server-password
MQTT_GATEWAY_PASSWORD=your-gateway-password
MQTT_BRIDGE_PASSWORD=your-bridge-password
MQTT_HOSTED_BROKER_ADDRESS=hosted-broker.example.com:1883
```
