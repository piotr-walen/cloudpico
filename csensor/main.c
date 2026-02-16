/**
 * BME280 I2C Driver for Raspberry Pi Pico
 * Using Bosch Sensortec BME280_SensorAPI (C, portable)
 *
 * NOTE: Ensure the device is capable of being driven at 3.3v NOT 5v. The Pico
 * GPIO (and therefore I2C) cannot be used at 5v.
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
 * Note: Using custom pins GP16 (SDA) and GP17 (SCL)
 *       Default pins would be GP4/GP5, but changed to GP16/GP17
 *
 * Alternative I2C pins (if needed):
 * - I2C0: GP0/GP1, GP4/GP5, GP8/GP9, GP12/GP13, GP16/GP17, GP20/GP21
 * - I2C1: GP2/GP3, GP6/GP7, GP10/GP11, GP14/GP15, GP18/GP19, GP22/GP23
 *
 * BME280 I2C Address:
 * - 0x76 if SDO pin is connected to GND
 * - 0x77 if SDO pin is connected to VCC
 */

#include <stdio.h>
#include "hardware/i2c.h"
#include "hardware/gpio.h"
#include "pico/binary_info.h"
#include "pico/stdlib.h"
#include "pico/time.h"
#include "bme280_pico_i2c.h"
#include "ble_advertise.h"
#include "flash_counter.h"

// BME280 I2C address (can be 0x76 or 0x77 depending on SDO pin)
#define BME280_ADDR _u(0x76)

// Custom I2C pin configuration
// Using GP16 (SDA) and GP17 (SCL) instead of default GP4/GP5
#define I2C_SDA_PIN 16 // GP16 (Pin 21) - SDA (Serial Data)
#define I2C_SCL_PIN 17 // GP17 (Pin 22) - SCL (Serial Clock)

// External LED on GP0 (Physical Pin 1)
#define LED_PIN 0 // GP0 (Pin 1) - External LED

#ifndef DEVICE_ID
#define DEVICE_ID 0x00000000
#endif

#ifndef POLL_INTERVAL_MS
#define POLL_INTERVAL_MS 10000
#endif

#ifndef ADVERTISEMENT_INTERVAL_MS
#define ADVERTISEMENT_INTERVAL_MS 5000
#endif


// Initialize external LED on GP0
static void led_init(void)
{
    gpio_init(LED_PIN);
    gpio_set_dir(LED_PIN, GPIO_OUT);
    gpio_put(LED_PIN, 0); // Start with LED off
}

// Set LED state (true = on, false = off)
static void led_set(bool on)
{
    gpio_put(LED_PIN, on);
}

// Print diagnostics and halt in an infinite noop loop (no return).
static void noop_loop(void)
{
    while (1)
    {
        __asm volatile("nop");
    }
}

int main()
{

    int8_t bme280_status;
    uint32_t run_count;

    // Initialize USB serial for debugging
    stdio_init_all();

    // Wait for USB serial to be ready (important for debugging)
    sleep_ms(3000);
    printf("Device ID = 0x%08X\n", DEVICE_ID);

    bool flash_counter_success = flash_counter_increment_and_store(&run_count);
    if (!flash_counter_success)
    {
        printf("ERROR: Flash counter increment/store failed, cannot continue\n");
        noop_loop();
    }

    printf("Run count = %lu\n", (unsigned long)run_count);

    bme280_status = bme280_sensor_init(I2C_SDA_PIN, I2C_SCL_PIN, BME280_ADDR, 100 * 1000);
    if (bme280_status != BME280_SENSOR_OK)
    {
        printf("ERROR: Failed to initialize BME280 sensor. Error code: %d\n", bme280_status);
        printf("ERROR: Check I2C connections (SDA=GP%d, SCL=GP%d) and sensor power\n", I2C_SDA_PIN, I2C_SCL_PIN);
        printf("ERROR: Program will exit. Press reset to retry.\n");
        noop_loop();
    }

    printf("BME280 initialized successfully.\n");
    printf("Starting sensor readings...\n\n");

    // Wait for sensor to stabilize
    sleep_ms(250);

    // Simple cycle: every POLL_INTERVAL_MS do read -> BLE init -> advertise -> BLE off
    absolute_time_t next_sensor_read = make_timeout_time_ms(POLL_INTERVAL_MS);

    while (1)
    {
        sleep_until(next_sensor_read);
        absolute_time_t cycle_start = get_absolute_time();

        // 1. Sensor reading
        float temperature, pressure, humidity;
        bme280_status = bme280_sensor_read(&temperature, &pressure, &humidity);
        if (bme280_status != BME280_SENSOR_OK)
        {
            printf("Failed to read sensor. Error code: %d\n", bme280_status);
            next_sensor_read = delayed_by_us(cycle_start, (uint64_t)POLL_INTERVAL_MS * 1000);
            continue;
        }

        printf("Temperature: %.2f C\n", temperature);
        printf("Pressure:    %.3f kPa\n", pressure);
        printf("Humidity:    %.2f %%\n", humidity);
        printf("---\n");

        sensor_data_t sensor_data = {
            .temperature = temperature,
            .pressure = pressure,
            .humidity = humidity
        };

        // 2. BLE init, wait ready (main loop), advertise, run (main loop), stop, deinit
        printf("BLE: Initializing...\n");
        int rc = ble_advertise_init(DEVICE_ID);
        if (rc != 0)
        {
            printf("BLE: Init failed (code %d), skipping advertisement\n", rc);
            next_sensor_read = delayed_by_us(cycle_start, (uint64_t)POLL_INTERVAL_MS * 1000);
            continue;
        }
        absolute_time_t adv_deadline = make_timeout_time_ms(ADVERTISEMENT_INTERVAL_MS);
        while (!ble_advertise_is_ready() && !time_reached(adv_deadline))
        {
            ble_advertise_poll_until(adv_deadline);
        }
        if (!ble_advertise_is_ready())
        {
            printf("BLE: Init timeout after %d ms, skipping advertisement\n", ADVERTISEMENT_INTERVAL_MS);
            ble_advertise_deinit();
            next_sensor_read = delayed_by_us(cycle_start, (uint64_t)POLL_INTERVAL_MS * 1000);
            continue;
        }
        ble_advertise_start(run_count, &sensor_data);
        while (!time_reached(adv_deadline))
        {
            ble_advertise_poll_until(adv_deadline);
        }
        ble_advertise_stop();
        ble_advertise_deinit();
        printf("BLE: Turned off after advertisement period\n");

        next_sensor_read = delayed_by_us(cycle_start, (uint64_t)POLL_INTERVAL_MS * 1000);
    }
}
