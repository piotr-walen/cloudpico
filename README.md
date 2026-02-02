# cloudpico

Overview

This project is a small, self-hosted weather station stack built around a Raspberry Pi Pico (firmware in TinyGo) publishing sensor telemetry over MQTT, with a Go backend that ingests readings into SQLite and exposes an HTTP API. It also serves a simple HTML web client for viewing current conditions and basic history.

Architecture

- Device (Raspberry Pi Pico + TinyGo)
Reads sensors at a fixed interval and publishes telemetry messages to MQTT.

- MQTT Broker (Mosquitto)
Receives telemetry and acts as the message backbone between device(s) and server.

- Backend (Go server)
Subscribes to telemetry topics, validates/parses payloads, stores readings in SQLite, provides an HTTP API, and serves the web UI.

- Reverse Proxy (Nginx)
Terminates TLS and proxies external HTTP requests to the Go server.

Components

firmware/ — TinyGo firmware for Raspberry Pi Pico (sensor reads + MQTT publish)

server/ — Go server (MQTT subscriber + HTTP API + SQLite persistence + HTML web client)

deploy/ — Docker Compose, Mosquitto/Nginx configs, deployment notes

docs/ — Topic conventions, payload schema, and project notes

Data Flow

- Pico samples sensors (temperature/humidity/pressure/etc.).

- Pico publishes telemetry to Mosquitto on a station-specific MQTT topic.

- Go server subscribes to telemetry topics and ingests messages.

- Readings are stored in SQLite for historical queries.

- Users open the web client (served by the Go server, via Nginx) which calls the API to render latest values and history.
