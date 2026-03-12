/**
 * BME280 I2C Adapter for Raspberry Pi Pico
 *
 * This file provides the I2C interface and BME280 initialization/read logic.
 * Only bme280_sensor_init() and bme280_sensor_read() are part of the public API.
 *
 */

#include <stdio.h>
#include "hardware/gpio.h"
#include "hardware/i2c.h"
#include "pico/stdlib.h"
#include "bme280.h"
#include "bme280_pico_i2c.h"

/**
 * I2C interface context (internal)
 */
struct bme280_pico_i2c_context {
    i2c_inst_t *i2c;
    uint8_t addr;
};

/* Internal state for single-sensor use */
static struct bme280_dev s_dev;
static struct bme280_pico_i2c_context s_i2c_ctx;
static uint32_t s_req_delay_us;

/**
 * @brief I2C read function for Bosch BME280 API (internal)
 */
static BME280_INTF_RET_TYPE bme280_pico_i2c_read(uint8_t reg_addr, uint8_t *reg_data, uint32_t len, void *intf_ptr)
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
 * @brief I2C write function for Bosch BME280 API (internal)
 */
static BME280_INTF_RET_TYPE bme280_pico_i2c_write(uint8_t reg_addr, const uint8_t *reg_data, uint32_t len, void *intf_ptr)
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
 * @brief Delay function for Bosch BME280 API (internal)
 */
static void bme280_pico_delay_us(uint32_t period, void *intf_ptr)
{
    (void)intf_ptr;  // Unused parameter
    sleep_us(period);
}

/**
 * @brief Initialize I2C interface for BME280 (internal)
 */
static void bme280_pico_i2c_init(struct bme280_pico_i2c_context *ctx,
                                  i2c_inst_t *i2c_instance,
                                  uint8_t i2c_addr,
                                  unsigned sda_pin,
                                  unsigned scl_pin,
                                  unsigned i2c_freq)
{
    ctx->i2c = i2c_instance;
    ctx->addr = i2c_addr;
    i2c_init(ctx->i2c, i2c_freq);
    gpio_set_function(sda_pin, GPIO_FUNC_I2C);
    gpio_set_function(scl_pin, GPIO_FUNC_I2C);
    gpio_pull_up(sda_pin);
    gpio_pull_up(scl_pin);
}

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
int8_t bme280_sensor_init(unsigned sda_pin, unsigned scl_pin, uint8_t i2c_addr, unsigned i2c_freq)
{
    int8_t rslt;
    struct bme280_settings settings;

    bme280_pico_i2c_init(&s_i2c_ctx, i2c_default, i2c_addr, sda_pin, scl_pin, i2c_freq);

    s_dev.intf = BME280_I2C_INTF;
    s_dev.read = bme280_pico_i2c_read;
    s_dev.write = bme280_pico_i2c_write;
    s_dev.delay_us = bme280_pico_delay_us;
    s_dev.intf_ptr = &s_i2c_ctx;

    rslt = bme280_init(&s_dev);
    if (rslt != BME280_OK)
        return rslt;

    settings.osr_p = BME280_OVERSAMPLING_1X;
    settings.osr_t = BME280_OVERSAMPLING_1X;
    settings.osr_h = BME280_OVERSAMPLING_1X;
    settings.filter = BME280_FILTER_COEFF_OFF;
    settings.standby_time = BME280_STANDBY_TIME_0_5_MS;

    rslt = bme280_set_sensor_settings(BME280_SEL_ALL_SETTINGS, &settings, &s_dev);
    if (rslt != BME280_OK)
        return rslt;

    rslt = bme280_cal_meas_delay(&s_req_delay_us, &settings);
    if (rslt != BME280_OK)
        return rslt;

    rslt = bme280_set_sensor_mode(BME280_POWERMODE_NORMAL, &s_dev);
    if (rslt != BME280_OK)
        return rslt;

    return BME280_OK;
}

/**
 * @brief Read compensated temperature, pressure and humidity.
 *
 * @param temperature_c  Output: temperature in °C (may be NULL)
 * @param pressure_kpa  Output: pressure in kPa (may be NULL)
 * @param humidity_pct  Output: relative humidity in % (may be NULL)
 * @return BME280_SENSOR_OK on success, non-zero on failure
 */
int8_t bme280_sensor_read(float *temperature_c, float *pressure_kpa, float *humidity_pct)
{
    int8_t rslt;
    struct bme280_data comp_data;

    rslt = bme280_get_sensor_data(BME280_ALL, &comp_data, &s_dev);
    if (rslt != BME280_OK)
        return rslt;

#ifdef BME280_DOUBLE_ENABLE
    if (temperature_c)
        *temperature_c = (float)comp_data.temperature;
    if (pressure_kpa)
        *pressure_kpa = (float)comp_data.pressure / 100.0f;  /* Pa -> hPa */
    if (humidity_pct)
        *humidity_pct = (float)comp_data.humidity;
#else
    if (temperature_c)
        *temperature_c = (float)comp_data.temperature / 100.0f;   /* 0.01°C -> °C */
    if (pressure_kpa)
        *pressure_kpa = (float)comp_data.pressure / 10000.0f;    /* Pa -> hPa */
    if (humidity_pct)
        *humidity_pct = (float)comp_data.humidity / 1000.0f;      /* milli-% -> % */
#endif

    return BME280_OK;
}
