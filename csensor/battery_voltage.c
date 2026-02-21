/**
 * DFRobot Gravity I2C 3.7V Li Battery Fuel Gauge (MAX17043) driver
 * for Raspberry Pi Pico (Pico 2 W) using Pico SDK.
 *
 * Uses I2C1 on GP18 (SDA) and GP19 (SCL). Voltage from VCELL register.
 */

#include <stdio.h>
#include <stdint.h>
#include "pico/stdlib.h"
#include "hardware/i2c.h"
#include "hardware/gpio.h"
#include "battery_voltage.h"

/* MAX17043 uses I2C1 so BME280 can keep I2C0 on GP16/GP17 */
#define BATTERY_I2C_INST    i2c1
#define BATTERY_SDA_PIN     18
#define BATTERY_SCL_PIN     19

/* MAX17043 register addresses (from datasheet) */
#define MAX17043_REG_VCELL  0x02u
#define MAX17043_REG_SOC    0x04u

/* VCELL: 12-bit value in upper bits, LSB = 1.25 mV (MAX17043) */
#define MAX17043_VCELL_LSB_MV    1.25f
#define MAX17043_VCELL_SHIFT     4

static bool battery_i2c_read_reg(uint8_t reg, uint8_t *buf, size_t len)
{
    int n = i2c_write_blocking(BATTERY_I2C_INST, BATTERY_FUEL_GAUGE_I2C_ADDR, &reg, 1, true);
    if (n != 1)
        return false;
    n = i2c_read_blocking(BATTERY_I2C_INST, BATTERY_FUEL_GAUGE_I2C_ADDR, buf, len, false);
    return (n == (int)len);
}

void battery_voltage_init(void)
{
    i2c_init(BATTERY_I2C_INST, BATTERY_FUEL_GAUGE_I2C_FREQ);
    gpio_set_function(BATTERY_SDA_PIN, GPIO_FUNC_I2C);
    gpio_set_function(BATTERY_SCL_PIN, GPIO_FUNC_I2C);
    gpio_pull_up(BATTERY_SDA_PIN);
    gpio_pull_up(BATTERY_SCL_PIN);
}

float read_battery_voltage(void)
{
    uint8_t buf[2];
    if (!battery_i2c_read_reg(MAX17043_REG_VCELL, buf, sizeof(buf)))
        return 0.0f;

    /* VCELL: MSB then LSB; 12-bit value in upper bits, 1.25 mV per LSB */
    uint16_t raw = (uint16_t)((buf[0] << 8) | buf[1]);
    uint16_t vcell = raw >> MAX17043_VCELL_SHIFT;
    return (float)vcell * MAX17043_VCELL_LSB_MV * 0.001f;
}

float read_battery_soc(void)
{
    uint8_t buf[2];
    if (!battery_i2c_read_reg(MAX17043_REG_SOC, buf, sizeof(buf)))
        return -1.0f;

    /* SOC: high byte = integer %, low byte = 1/256 % */
    float percent = (float)buf[0] + (float)buf[1] / 256.0f;
    return percent;
}
