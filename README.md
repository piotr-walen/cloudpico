# Cloudpico

Self-hosted weather station stack built around **battery-powered Raspberry Pi Pico 2** devices that broadcast sensor telemetry via **BLE advertisements (connectionless)**. **C firmware** (pico-sdk) on the Pico; a **Linux host** scans BLE ads, validates and parses payloads, and republishes to **MQTT**; a **Go API** ingests into **SQLite** and exposes an **HTTP API**. A **React web app** (in `web/`) talks to the API for current conditions and history.

## Architecture

**Sensor (MCU w/ BLE)**
Reads sensors at a fixed interval, encodes readings into a compact binary payload, and broadcasts them via **BLE advertising**. Devices never connect or pair; they wake → measure → advertise briefly → sleep.

**Gateway / Linux host**
Runs a BLE scanner that passively listens for station advertisements, deduplicates and validates messages (sequence numbers, timestamps, optional CRC), tracks per-station “last seen” state, and republishes decoded telemetry into MQTT topics.

**MQTT (Mosquitto)**  
Two roles: **local broker** (gateway publishes here; bridges to central broker) and **remote/hosted broker** (central; API subscribes here). See `docs/mosquitto.md` and `local-broker/`, `remote-broker/`.

**API (Go)**  
Subscribes to telemetry on the remote broker, validates and parses payloads, stores readings in SQLite, and provides an HTTP API. Serves the built web app from `api/static` (or use the Vite dev server for `web/` during development).

**Web**  
React + TypeScript + Vite frontend for viewing current conditions and history; consumes the API.

## Components

| Directory | Description |
|-----------|-------------|
| `api/` | Go server: MQTT subscriber, HTTP API, SQLite, static file serving |
| `web/` | React frontend (Vite); build output can be copied to `api/static` |
| `gateway/` | BLE scanner → MQTT bridge (station health, deduplication) |
| `csensor/` | C firmware for Raspberry Pi Pico 2 (pico-sdk): sensors + BLE advertising + sleep |
| `local-broker/` | Mosquitto config for the broker next to the gateway (bridge to remote) |
| `remote-broker/` | Mosquitto config for the central/hosted broker (API subscribes here) |
| `docs/` | BLE payload schema, MQTT topics, broker setup (e.g. `docs/mosquitto.md`) |

Docker Compose files are at the repo root: `docker-compose.dev.yaml` (local + remote brokers for dev), `docker-compose.remote.yaml`, `docker-compose.local.yaml`.

## Data flow

1. Pico 2 wakes on interval, samples sensors, packs **station_id + sequence + battery + sensor values** into a compact payload, and broadcasts in BLE advertisements.
2. Gateway on the Linux host scans BLE ads, filters and validates, then publishes telemetry to the **local** Mosquitto broker on station-specific topics.
3. Local broker bridges those topics to the **remote** broker.
4. API subscribes to telemetry on the remote broker and ingests messages into SQLite.
5. Web app calls the API to show latest values and history.

