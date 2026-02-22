/**
 * BLE Advertising Module for CloudPico
 * 
 * Advertises BME280 sensor data via BLE manufacturer data in the format
 * expected by the gateway:
 * - Magic: 0x01, 0xD0 (2 bytes)
 * - device_id: uint8 (lower 8 bits of uint32, 0–255) (1 byte)
 * - run_count16: uint16 (little-endian) (2 bytes)
 * - seq24: uint24 (little-endian) (3 bytes)
 * - temperature: int24 LE scaled by 100 (0.01°C resolution) (3 bytes)
 * - pressure: uint24 LE scaled by 10 (0.1 kPa resolution) (3 bytes)
 * - humidity: uint24 LE scaled by 100 (0.01% resolution) (3 bytes)
 * - battery: uint16 LE scaled by 1000 (0.001V resolution) (2 bytes)
 * - battery_soc: uint16 LE scaled by 100 (0.01% resolution), 0xFFFF = invalid (2 bytes)
 * Total: 21 bytes
 *
 * device_id and run_count are uint32_t internally; only lower 8 and 16 bits are transmitted.
 * reading_id: lower 24 bits transmitted.
 */

#ifndef BLE_ADVERTISE_H
#define BLE_ADVERTISE_H

#include <stdint.h>
#include <stdbool.h>
#include "pico/time.h"

// Company ID used for manufacturer data (matches gateway filter)
#define BLE_COMPANY_ID 0xFFFF

// Magic bytes that identify our sensor payload
#define BLE_MAGIC_0 0x01
#define BLE_MAGIC_1 0xD0

// Sensor data structure
typedef struct {
    float temperature;  // Celsius
    float pressure;     // kPa
    float humidity;     // %RH
    float battery;     // Battery voltage in volts
    float battery_soc; // State of charge 0–100%, or <0 if invalid/unknown
} sensor_data_t;

/**
 * Initialize BLE advertising (powers on BTstack and starts HCI).
 * Poll with ble_advertise_is_ready() and ble_advertise_poll_until() in the main loop until ready.
 * @param device_id Unique device identifier (uint32_t, only lower 16 bits will be advertised)
 * @return 0 on success, negative on error
 */
int ble_advertise_init(uint32_t device_id);

/**
 * Check if BLE is initialized and ready to advertise.
 * @return true if initialized and ready, false otherwise
 */
bool ble_advertise_is_ready(void);

/**
 * Drive the BLE/CYW43 async context once, optionally waiting until deadline.
 * Call this in the main loop while waiting (e.g. until is_ready() or until advertisement interval ends).
 * @param until Absolute time until which to wait for work; use make_timeout_time_ms() to build
 */
void ble_advertise_poll_until(absolute_time_t until);

/**
 * Start BLE advertisement with sensor data (updates payload and enables advertising).
 * @param run_count Current run count
 * @param data Sensor readings (temperature, pressure, humidity)
 * @return 0 on success, negative on error
 */
int ble_advertise_start(uint32_t run_count, sensor_data_t *data);

/**
 * Stop BLE advertisement (disable advertising; BLE remains initialized).
 */
void ble_advertise_stop(void);

/**
 * Deinitialize BLE advertising (stops advertising and powers off Bluetooth).
 */
void ble_advertise_deinit(void);

#endif // BLE_ADVERTISE_H
