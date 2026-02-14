/**
 * BME280 I2C Adapter for Raspberry Pi Pico
 * 
 * This file provides the I2C interface functions required by the Bosch Sensortec
 * BME280 Sensor API to work with Raspberry Pi Pico's I2C hardware.
 * 
 * PIN CONNECTIONS (Raspberry Pi Pico):
 * ====================================
 * Pico Pin  | GPIO  | Function | BME280 Pin
 * ----------|-------|----------|------------
 * Pin 21    | GP16  | SDA      | SDA
 * Pin 22    | GP17  | SCL      | SCL
 * Pin 36    | 3V3   | Power    | VCC/VIN
 * Pin 38    | GND   | Ground   | GND
 * 
 * BME280 I2C Address:
 * - 0x76 if SDO pin is connected to GND
 * - 0x77 if SDO pin is connected to VCC
 */

#include <stdio.h>
#include "hardware/i2c.h"
#include "pico/stdlib.h"
#include "bme280.h"

// BME280 I2C address (can be 0x76 or 0x77 depending on SDO pin)
#define BME280_ADDR _u(0x76)

// Custom I2C pin configuration
// Using GP16 (SDA) and GP17 (SCL) instead of default GP4/GP5
#define I2C_SDA_PIN 16   // GP16 (Pin 21) - SDA (Serial Data)
#define I2C_SCL_PIN 17   // GP17 (Pin 22) - SCL (Serial Clock)

/**
 * Structure to hold I2C interface context
 */
struct bme280_pico_i2c_context {
    i2c_inst_t *i2c;
    uint8_t addr;
};

/**
 * @brief I2C read function for Bosch BME280 API
 * 
 * This function is called by the Bosch API to read data from the BME280 sensor
 * via I2C. It adapts the Bosch API's function signature to Raspberry Pi Pico's
 * I2C functions.
 * 
 * @param reg_addr Register address to read from
 * @param reg_data Buffer to store read data
 * @param len Number of bytes to read
 * @param intf_ptr Pointer to interface context (bme280_pico_i2c_context)
 * @return BME280_INTF_RET_SUCCESS on success, non-zero on failure
 */
BME280_INTF_RET_TYPE bme280_pico_i2c_read(uint8_t reg_addr, uint8_t *reg_data, uint32_t len, void *intf_ptr)
{
    struct bme280_pico_i2c_context *ctx = (struct bme280_pico_i2c_context *)intf_ptr;
    
    if (ctx == NULL || reg_data == NULL) {
        return BME280_E_NULL_PTR;
    }
    
    // Write register address, then read data
    int ret = i2c_write_blocking(ctx->i2c, ctx->addr, &reg_addr, 1, true);  // true = keep master control
    if (ret != 1) {
        return BME280_E_COMM_FAIL;
    }
    
    ret = i2c_read_blocking(ctx->i2c, ctx->addr, reg_data, len, false);  // false = finished with bus
    if (ret != (int)len) {
        return BME280_E_COMM_FAIL;
    }
    
    return BME280_INTF_RET_SUCCESS;
}

/**
 * @brief I2C write function for Bosch BME280 API
 * 
 * This function is called by the Bosch API to write data to the BME280 sensor
 * via I2C. It adapts the Bosch API's function signature to Raspberry Pi Pico's
 * I2C functions.
 * 
 * @param reg_addr Register address to write to
 * @param reg_data Buffer containing data to write
 * @param len Number of bytes to write
 * @param intf_ptr Pointer to interface context (bme280_pico_i2c_context)
 * @return BME280_INTF_RET_SUCCESS on success, non-zero on failure
 */
BME280_INTF_RET_TYPE bme280_pico_i2c_write(uint8_t reg_addr, const uint8_t *reg_data, uint32_t len, void *intf_ptr)
{
    struct bme280_pico_i2c_context *ctx = (struct bme280_pico_i2c_context *)intf_ptr;
    
    if (ctx == NULL || reg_data == NULL) {
        return BME280_E_NULL_PTR;
    }
    
    // The Bosch API prepares the buffer differently for single vs burst writes:
    // - Single write: reg_data contains just the data byte
    // - Burst write: reg_data contains data[0], then interleaved reg_addr[1], data[1], reg_addr[2], data[2], ...
    // For I2C, we always write: [reg_addr] + [reg_data buffer]
    
    // Allocate buffer for register address + data
    uint8_t buf[21];  // Max 20 bytes data + 1 byte reg_addr
    if (len > 20) {
        return BME280_E_INVALID_LEN;
    }
    
    buf[0] = reg_addr;
    for (uint32_t i = 0; i < len; i++) {
        buf[i + 1] = reg_data[i];
    }
    
    int ret = i2c_write_blocking(ctx->i2c, ctx->addr, buf, len + 1, false);
    if (ret != (int)(len + 1)) {
        return BME280_E_COMM_FAIL;
    }
    
    return BME280_INTF_RET_SUCCESS;
}

/**
 * @brief Delay function for Bosch BME280 API
 * 
 * This function provides microsecond delays required by the Bosch API.
 * 
 * @param period Delay period in microseconds
 * @param intf_ptr Pointer to interface context (unused)
 */
void bme280_pico_delay_us(uint32_t period, void *intf_ptr)
{
    (void)intf_ptr;  // Unused parameter
    sleep_us(period);
}

/**
 * @brief Initialize I2C interface for BME280
 * 
 * This function initializes the Raspberry Pi Pico I2C hardware and sets up
 * the interface context for the Bosch BME280 API.
 * 
 * @param ctx Pointer to I2C context structure (output)
 * @param i2c_instance I2C instance to use (i2c0 or i2c1)
 * @param i2c_addr I2C address of BME280 (0x76 or 0x77)
 * @param sda_pin GPIO pin for SDA
 * @param scl_pin GPIO pin for SCL
 * @param i2c_freq I2C frequency in Hz (e.g., 100000 for 100 kHz)
 */
void bme280_pico_i2c_init(struct bme280_pico_i2c_context *ctx,
                          i2c_inst_t *i2c_instance,
                          uint8_t i2c_addr,
                          uint sda_pin,
                          uint scl_pin,
                          uint i2c_freq)
{
    ctx->i2c = i2c_instance;
    ctx->addr = i2c_addr;
    
    // Initialize I2C
    i2c_init(ctx->i2c, i2c_freq);
    gpio_set_function(sda_pin, GPIO_FUNC_I2C);
    gpio_set_function(scl_pin, GPIO_FUNC_I2C);
    gpio_pull_up(sda_pin);
    gpio_pull_up(scl_pin);
}
