# Mosquitto MQTT Broker Configuration

## Configuration Files

Example configuration files are available in the `docs/` directory:
- `docs/mosquitto.conf.example` - Example Mosquitto broker configuration
- `docs/mosquitto.acl.example` - Example Access Control List (ACL)

Copy these to the `mosquitto/` directory:
```bash
cp docs/mosquitto.conf.example mosquitto/mosquitto.conf
cp docs/mosquitto.acl.example mosquitto/acl
```

## Setting Up Authentication

### Create password file with server user:
```bash
mosquitto_passwd -c -b mosquitto/passwd server "your-server-password"
```

### Add gateway user:
```bash
mosquitto_passwd -b mosquitto/passwd gateway "your-gateway-password"
```

### Update existing password file:
```bash
# Update server password
mosquitto_passwd -b mosquitto/passwd server "new-server-password"

# Update gateway password
mosquitto_passwd -b mosquitto/passwd gateway "new-gateway-password"
```

## Environment Variables

After generating passwords, set them as environment variables for docker-compose:

```bash
export MQTT_SERVER_PASSWORD="your-server-password"
export MQTT_GATEWAY_PASSWORD="your-gateway-password"
```

Or add them to a `.env` file in the project root:

```env
MQTT_SERVER_PASSWORD=your-server-password
MQTT_GATEWAY_PASSWORD=your-gateway-password
```
