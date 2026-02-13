/**
 * BME280 I2C Adapter for Raspberry Pi Pico
 * 
 * Header file for I2C interface adapter between Bosch Sensortec BME280 Sensor API
 * and Raspberry Pi Pico's I2C hardware.
 */

#ifndef _BME280_PICO_I2C_H
#define _BME280_PICO_I2C_H

#include "hardware/i2c.h"
#include "bme280.h"

/**
 * Structure to hold I2C interface context
 */
struct bme280_pico_i2c_context {
    i2c_inst_t *i2c;
    uint8_t addr;
};

/**
 * @brief I2C read function for Bosch BME280 API
 */
BME280_INTF_RET_TYPE bme280_pico_i2c_read(uint8_t reg_addr, uint8_t *reg_data, uint32_t len, void *intf_ptr);

/**
 * @brief I2C write function for Bosch BME280 API
 */
BME280_INTF_RET_TYPE bme280_pico_i2c_write(uint8_t reg_addr, const uint8_t *reg_data, uint32_t len, void *intf_ptr);

/**
 * @brief Delay function for Bosch BME280 API
 */
void bme280_pico_delay_us(uint32_t period, void *intf_ptr);

/**
 * @brief Initialize I2C interface for BME280
 */
void bme280_pico_i2c_init(struct bme280_pico_i2c_context *ctx,
                          i2c_inst_t *i2c_instance,
                          uint8_t i2c_addr,
                          uint sda_pin,
                          uint scl_pin,
                          uint i2c_freq);

#endif /* _BME280_PICO_I2C_H */
