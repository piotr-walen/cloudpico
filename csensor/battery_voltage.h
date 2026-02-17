#ifndef BATTERY_VOLTAGE_H
#define BATTERY_VOLTAGE_H

#include <stdint.h>

/**
 * Initialize ADC for battery voltage reading.
 * Sets up GPIO26 as ADC0 input.
 */
void battery_voltage_init(void);

/**
 * Read the battery voltage using ADC.
 * 
 * Uses a voltage divider (100k/100k) to measure battery voltage.
 * Call battery_voltage_init() first before using this function.
 * 
 * @return Battery voltage in volts
 */
float read_battery_voltage(void);

#endif // BATTERY_VOLTAGE_H
