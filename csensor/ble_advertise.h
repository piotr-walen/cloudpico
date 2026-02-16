/**
 * BLE Advertising Module for CloudPico
 * 
 * Advertises BME280 sensor data via BLE manufacturer data in the format
 * expected by the gateway:
 * - Magic: 0x01, 0xD0
 * - device_id: uint16 (little-endian, lower 16 bits of uint32)
 * - run_count: uint32 (little-endian)
 * - reading_id: uint32 (little-endian)
 * - temperature: float32 (little-endian)
 * - pressure: float32 (little-endian)
 * - humidity: float32 (little-endian)
 * 
 * Note: device_id is stored and passed as uint32_t internally, but only
 * the lower 16 bits (0xFFFF) are transmitted in the BLE advertisement.
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
