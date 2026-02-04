This project is a self-hosted weather station stack built around battery-powered Raspberry Pi Pico 2 devices (firmware in TinyGo) that broadcast sensor telemetry using BLE advertisements (connectionless). A Raspberry Pi 5 (or other Linux host) continuously scans for these advertisements, validates and parses payloads, republishes telemetry into MQTT, and a Go backend ingests readings into SQLite and exposes an HTTP API. The backend also serves a simple HTML web client for viewing current conditions and basic history.

Architecture

Device (Raspberry Pi Pico 2 + TinyGo)
Reads sensors at a fixed interval, encodes readings into a compact binary payload, and broadcasts them via BLE advertising. Devices never connect or pair; they wake → measure → advertise briefly → sleep.

Collector / Gateway (Raspberry Pi 5 / Linux host)
Runs a BLE scanner that passively listens for station advertisements, deduplicates and validates messages (sequence numbers, timestamps, optional CRC), tracks per-station “last seen” state, and republishes decoded telemetry into MQTT topics.

MQTT Broker (Mosquitto)
Receives telemetry from the gateway and acts as the message backbone between gateway and server (and optionally other consumers).

Backend (Go server)
Subscribes to telemetry topics, validates and parses payloads, stores readings in SQLite, provides an HTTP API, and serves the web UI.

Components

- firmware/ — TinyGo firmware for Raspberry Pi Pico 2 (sensor reads + BLE advertising payload + sleep strategy)

- gateway/ — BLE scanner/collector service (BLE → MQTT bridge, station health, deduplication)

- server/ — Go server (MQTT subscriber + HTTP API + SQLite persistence + HTML web client)

- deploy/ — Docker Compose, Mosquitto configs, deployment notes

- docs/ — BLE payload schema, station identity conventions, MQTT topic conventions, API notes

Data Flow

- Pico 2 wakes on interval and samples sensors (temperature, humidity, pressure, etc.).

- Pico 2 packs readings into a compact payload including station_id + sequence number + battery + sensor values and broadcasts it in BLE advertisements for a short window.

- Pi 5 passively scans BLE advertisements, filters by service/manufacturer data, validates payloads, updates station “last seen” and health state, and republishes telemetry to Mosquitto on a station-specific topic.

- Go server subscribes to telemetry topics and ingests messages.

- Readings are stored in SQLite for historical queries.

- Users open the web client (served directly by the Go server) which calls the API to render latest values and history.
