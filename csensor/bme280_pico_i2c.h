/**
 * BME280 I2C Adapter for Raspberry Pi Pico
 *
 * Public API: sensor init and sensor read only.
 */

#ifndef _BME280_PICO_I2C_H
#define _BME280_PICO_I2C_H

#include <stdint.h>

/** Success return value for init/read (0). Non-zero = Bosch error code. */
#define BME280_SENSOR_OK 0

/**
 * @brief Initialize BME280 sensor (I2C, settings, normal mode).
 *
 * Call once before bme280_sensor_read(). Uses default oversampling x1,
 * filter off, standby 0.5 ms.
 *
 * @param sda_pin   GPIO pin for I2C SDA
 * @param scl_pin   GPIO pin for I2C SCL
 * @param i2c_addr  BME280 I2C address (0x76 or 0x77)
 * @param i2c_freq  I2C frequency in Hz (e.g. 100000)
 * @return BME280_SENSOR_OK on success, non-zero Bosch error code on failure
 */
int8_t bme280_sensor_init(unsigned sda_pin, unsigned scl_pin, uint8_t i2c_addr, unsigned i2c_freq);

/**
 * @brief Read compensated temperature, pressure and humidity.
 *
 * @param temperature_c  Output: temperature in °C (may be NULL)
 * @param pressure_kpa   Output: pressure in kPa (may be NULL)
 * @param humidity_pct  Output: relative humidity in % (may be NULL)
 * @return BME280_SENSOR_OK on success, non-zero on failure
 */
int8_t bme280_sensor_read(float *temperature_c, float *pressure_kpa, float *humidity_pct);

#endif /* _BME280_PICO_I2C_H */
