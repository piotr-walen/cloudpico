/**
 * BLE Advertising Module for CloudPico
 * 
 * Advertises BME280 sensor data via BLE manufacturer data in the format
 * expected by the gateway:
 * - Magic: 0x01, 0xD0
 * - device_id: uint32 (little-endian)
 * - reading_id: uint32 (little-endian)
 * - temperature: float32 (little-endian)
 * - pressure: float32 (little-endian)
 * - humidity: float32 (little-endian)
 */

#ifndef BLE_ADVERTISE_H
#define BLE_ADVERTISE_H

#include <stdint.h>
#include <stdbool.h>

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
 * Initialize BLE advertising
 * @param device_id Unique device identifier (will be advertised in manufacturer data)
 * @return 0 on success, negative on error
 */
int ble_advertise_init(uint32_t device_id);

/**
 * Update advertisement with new sensor data
 * @param data Sensor readings (temperature, pressure, humidity)
 * @return 0 on success, negative on error
 */
int ble_advertise_update(sensor_data_t *data);

/**
 * Deinitialize BLE advertising
 */
void ble_advertise_deinit(void);

/**
 * Check if BLE is initialized and ready
 * @return true if initialized, false otherwise
 */
bool ble_advertise_is_ready(void);

#endif // BLE_ADVERTISE_H
