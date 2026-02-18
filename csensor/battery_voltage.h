#ifndef BATTERY_VOLTAGE_H
#define BATTERY_VOLTAGE_H

#include <stdint.h>

/* ---------------------------------------------------------------------------
 * Board-specific configuration (adjust for hardware revisions)
 * ---------------------------------------------------------------------------
 *
 * HARDWARE ASSUMPTIONS:
 * - ADC: RP2040 internal 12-bit ADC (0..ADC_MAX), single-ended on ADC0 (GPIO26).
 * - ADC reference: 3V3 rail (VBUS-derived or regulator). Accuracy depends on
 *   board supply; typical ~3.3V. For precision use external reference and
 *   set BATTERY_ADC_REF_V accordingly.
 * - Voltage divider: resistor divider from battery to ADC. Ratio must match
 *   BATTERY_DIVIDER_RATIO.
 *   Example: R1 (to battery) = 100k, R2 (to GND) = 100k => ratio = (R1+R2)/R2 = 2.0.
 *   Different boards: change R1/R2 and update BATTERY_DIVIDER_RATIO.
 * - Sampling: multiple samples with inter-sample delay to reduce noise;
 *   tune count/delay if needed for different ADC or filter requirements.
 * --------------------------------------------------------------------------- */

/** ADC reference voltage (V). Source: 3V3 rail on Pico. */
#define BATTERY_ADC_REF_V          (3.3f)

/** 12-bit ADC maximum count (inclusive). */
#define BATTERY_ADC_MAX_COUNT      (4095)

/** Voltage divider ratio = V_battery / V_adc. E.g. 100k/100k => 2.0. */
#define BATTERY_DIVIDER_RATIO      (2.0f)

/** Number of ADC samples to average (noise reduction). */
#define BATTERY_ADC_SAMPLES        (32)

/** Delay between samples in microseconds (settling / anti-aliasing). */
#define BATTERY_ADC_SAMPLE_DELAY_US (200)

/**
 * Initialize ADC for battery voltage reading.
 * Sets up GPIO26 as ADC0 input.
 */
void battery_voltage_init(void);

/**
 * Read the battery voltage using ADC.
 *
 * Uses the voltage divider and ADC parameters defined above.
 * Call battery_voltage_init() first before using this function.
 *
 * @return Battery voltage in volts
 */
float read_battery_voltage(void);

#endif // BATTERY_VOLTAGE_H
