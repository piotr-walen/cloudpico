# Example secrets for Mosquitto

Copy these files into a `secrets/` directory at the **repository root** and replace the placeholder values with real passwords. The `secrets/` directory is gitignored.

```bash
cp -r secrets.example secrets
# Edit secrets/mqtt_* with your passwords
chmod 600 secrets/mqtt_*
```

| File | Used by | Purpose |
|------|---------|---------|
| `mqtt_api_password` | Remote / dev broker | Password for user `api` (API subscribes to telemetry) |
| `mqtt_bridge_password` | Remote / dev broker | Password for user `bridge` (local broker bridge) |
| `mqtt_gateway_password` | Local / dev broker | Password for user `gateway` (gateway publishes telemetry) |

See the main [README.md](../README.md#secrets) and [docs/mosquitto.md](../docs/mosquitto.md) for details.
