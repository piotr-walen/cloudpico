#ifndef BATTERY_VOLTAGE_H
#define BATTERY_VOLTAGE_H

#include <stdint.h>

/* ---------------------------------------------------------------------------
 * DFRobot Gravity I2C 3.7V Li Battery Fuel Gauge (MAX17043)
 * ---------------------------------------------------------------------------
 *
 * HARDWARE:
 * - Module: DFRobot Gravity I2C 3.7V Li Battery Fuel Gauge (SKU DFR0563).
 * - IC: MAX17043 (1-cell LiPo fuel gauge).
 * - I2C: 7-bit address 0x36. Uses I2C1 on Pico to avoid conflict with BME280 on I2C0.
 *
 * PINS (Raspberry Pi Pico 2 W):
 * - GP18 = I2C1 SDA (connect to module SDA)
 * - GP19 = I2C1 SCL (connect to module SCL)
 *
 * Voltage is read from the VCELL register; resolution 1.25 mV.
 * Optional: SOC (state of charge) can be read for percentage.
 * --------------------------------------------------------------------------- */

/** I2C address of the MAX17043 (7-bit). */
#define BATTERY_FUEL_GAUGE_I2C_ADDR   (0x36u)

/** I2C clock frequency (Hz). */
#define BATTERY_FUEL_GAUGE_I2C_FREQ   (100000u)

/**
 * Initialize I2C for the DFRobot fuel gauge (MAX17043).
 * Uses I2C1 on GP18 (SDA) and GP19 (SCL).
 * Call once before read_battery_voltage() or read_battery_soc().
 */
void battery_voltage_init(void);

/**
 * Read battery voltage from the fuel gauge (VCELL).
 *
 * @return Battery voltage in volts, or 0.0f on I2C error
 */
float read_battery_voltage(void);

/**
 * Read state of charge from the fuel gauge (optional).
 *
 * @return SOC in percent (0.0f .. 100.0f), or -1.0f on I2C error
 */
float read_battery_soc(void);

#endif // BATTERY_VOLTAGE_H
