# Mosquitto MQTT Broker Configuration

## Broker Layout

The project uses two Mosquitto broker roles:

- **Local broker** (`local-broker/`) – Runs near the gateway. The gateway publishes to this broker; it bridges selected topics to the remote broker.
- **Remote broker** (`remote-broker/`) – Central/hosted broker. Local brokers bridge to it; the API subscribes here.

Configuration files (`mosquitto.conf`, `acl`) are in `local-broker/` and `remote-broker/`. Passwords are supplied via **Docker secrets**; an entrypoint script generates the Mosquitto `password_file` at container start (Mosquitto does not read Docker secrets directly).

## Setting Up Authentication (Docker Secrets)

Create plain-text password files under `secrets/` (the directory is gitignored). The entrypoint reads these and builds the Mosquitto passwd file at startup.

### Remote broker (hosted)

Users: `bridge` (local broker bridge), `api` (API subscriber). Create:

- `secrets/mqtt_api_password` – password for user `api`
- `secrets/mqtt_bridge_password` – password for user `bridge`

```bash
mkdir -p secrets
echo -n "your-api-password"    > secrets/mqtt_api_password
echo -n "your-bridge-password" > secrets/mqtt_bridge_password
chmod 600 secrets/mqtt_api_password secrets/mqtt_bridge_password
```

### Local broker

User: `gateway`. Create:

- `secrets/mqtt_gateway_password` – password for user `gateway`

```bash
echo -n "your-gateway-password" > secrets/mqtt_gateway_password
chmod 600 secrets/mqtt_gateway_password
```

### Updating passwords

Change the contents of the files in `secrets/` and restart the broker container(s). The passwd file is regenerated on every start.

## Environment Variables

For the **API** (`api/.env`), set at least:

```env
MQTT_BROKER=localhost
MQTT_PORT=1885
MQTT_USERNAME=api
MQTT_PASSWORD=your-api-password
MQTT_TOPIC=stations/+/telemetry
```

For dev, the remote broker is on host port 1885 and the local broker on 1884 (see `docker-compose.dev.yaml`). Use the same secrets as above; ensure `secrets/mqtt_api_password`, `secrets/mqtt_bridge_password`, and `secrets/mqtt_gateway_password` exist. After changing any secret file, restart the broker container(s).