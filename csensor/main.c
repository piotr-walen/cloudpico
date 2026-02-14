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
#include "bme280.h"
#include "bme280_pico_i2c.h"
#include "ble_advertise.h"

// Pico W devices need CYW43 for BLE
#ifdef CYW43_WL_GPIO_LED_PIN
#include "pico/cyw43_arch.h"
#include "pico/async_context.h"
#endif

// BME280 I2C address (can be 0x76 or 0x77 depending on SDO pin)
#define BME280_ADDR _u(0x76)

// Custom I2C pin configuration
// Using GP16 (SDA) and GP17 (SCL) instead of default GP4/GP5
#define I2C_SDA_PIN 16   // GP16 (Pin 21) - SDA (Serial Data)
#define I2C_SCL_PIN 17   // GP17 (Pin 22) - SCL (Serial Clock)



// External LED on GP0 (Physical Pin 1)
#define LED_PIN 0   // GP0 (Pin 1) - External LED

#ifndef DEVICE_ID
#define DEVICE_ID 0x00000000
#endif

#ifndef POLL_INTERVAL_MS
#define POLL_INTERVAL_MS 10000
#endif

// Initialize external LED on GP0
static void led_init(void) {
    gpio_init(LED_PIN);
    gpio_set_dir(LED_PIN, GPIO_OUT);
    gpio_put(LED_PIN, 0);  // Start with LED off
}

// Set LED state (true = on, false = off)
static void led_set(bool on) {
    gpio_put(LED_PIN, on);
}

// Print diagnostics and halt in an infinite noop loop (no return).
static void noop_loop(void) {
    while (1) {
        __asm volatile ("nop");
    }
}

int main() {
    stdio_init_all();
    
    // Wait for USB serial to be ready (important for debugging)
    sleep_ms(3000);

#if !defined(i2c_default)
    #warning i2c / bme280_i2c example requires a board with I2C support
    puts("I2C not available");
    return 0;
#else
    int8_t rslt;
    struct bme280_dev dev;
    struct bme280_data comp_data;
    struct bme280_pico_i2c_context i2c_ctx;
    uint32_t req_delay;

    // Initialize external LED on GP0
    led_init();
    printf("LED initialized on GP%d (Pin 1)\n", LED_PIN);

    // Useful information for picotool
    bi_decl(bi_2pins_with_func(I2C_SDA_PIN, I2C_SCL_PIN, GPIO_FUNC_I2C));
    bi_decl(bi_1pin_with_name(LED_PIN, "External LED"));
    bi_decl(bi_program_description("BME280 I2C example using Bosch Sensortec API for Raspberry Pi Pico"));

    printf("Hello, BME280! Using Bosch Sensortec BME280_SensorAPI\n");
    printf("Initializing I2C on GP%d (SDA) and GP%d (SCL)...\n", I2C_SDA_PIN, I2C_SCL_PIN);

    // Initialize I2C interface
    bme280_pico_i2c_init(&i2c_ctx, i2c_default, BME280_ADDR, I2C_SDA_PIN, I2C_SCL_PIN, 100 * 1000);

    // Initialize BME280 device structure
    dev.intf = BME280_I2C_INTF;
    dev.read = bme280_pico_i2c_read;
    dev.write = bme280_pico_i2c_write;
    dev.delay_us = bme280_pico_delay_us;
    dev.intf_ptr = &i2c_ctx;

    // Initialize the sensor
    rslt = bme280_init(&dev);
    if (rslt != BME280_OK) {
        printf("ERROR: Failed to initialize BME280 sensor. Error code: %d\n", rslt);
        printf("ERROR: Check I2C connections (SDA=GP%d, SCL=GP%d) and sensor power\n", I2C_SDA_PIN, I2C_SCL_PIN);
        printf("ERROR: Program will exit. Press reset to retry.\n");
        noop_loop();
    }

    printf("BME280 initialized successfully. Chip ID: 0x%02X\n", dev.chip_id);

    // Configure sensor settings
    // Recommended settings: oversampling x1 for all sensors, filter off, standby 0.5ms
    struct bme280_settings settings;
    settings.osr_p = BME280_OVERSAMPLING_1X;
    settings.osr_t = BME280_OVERSAMPLING_1X;
    settings.osr_h = BME280_OVERSAMPLING_1X;
    settings.filter = BME280_FILTER_COEFF_OFF;
    settings.standby_time = BME280_STANDBY_TIME_0_5_MS;

    rslt = bme280_set_sensor_settings(BME280_SEL_ALL_SETTINGS, &settings, &dev);
    if (rslt != BME280_OK) {
        printf("ERROR: Failed to set sensor settings. Error code: %d\n", rslt);
        printf("ERROR: Program will exit. Press reset to retry.\n");
        noop_loop();
    }

    // Calculate measurement delay
    rslt = bme280_cal_meas_delay(&req_delay, &settings);
    if (rslt != BME280_OK) {
        printf("ERROR: Failed to calculate measurement delay. Error code: %d\n", rslt);
        printf("ERROR: Program will exit. Press reset to retry.\n");
        noop_loop();
    }

    // Set sensor to normal mode
    rslt = bme280_set_sensor_mode(BME280_POWERMODE_NORMAL, &dev);
    if (rslt != BME280_OK) {
        printf("ERROR: Failed to set sensor mode. Error code: %d\n", rslt);
        printf("ERROR: Program will exit. Press reset to retry.\n");
        noop_loop();
    }

    printf("Sensor configured. Measurement delay: %lu us\n", req_delay);
    
    // Initialize BLE advertising (only on Pico W)
    #ifdef CYW43_WL_GPIO_LED_PIN
    printf("Initializing BLE advertising...\n");
    int rc = ble_advertise_init(DEVICE_ID);
    if (rc != 0) {
        printf("Warning: BLE advertising initialization failed (code: %d). Continuing without BLE.\n", rc);
    } else {
        printf("BLE advertising initialized successfully.\n");
    }
    #else
    printf("Note: BLE not available (requires Pico W). Continuing with sensor readings only.\n");
    #endif
    
    printf("Starting sensor readings...\n\n");

    // Wait for sensor to stabilize
    sleep_ms(250);

    // Timing for sensor readings (every 1 second)
    absolute_time_t next_sensor_read = make_timeout_time_ms(POLL_INTERVAL_MS);
    bool led_state = false;

    while (1) {
        // Poll async context for BLE events (required for Pico W with BLE)
        // For Pico W, cyw43_arch_async_context() is available
        #ifdef CYW43_WL_GPIO_LED_PIN
        async_context_poll(cyw43_arch_async_context());
        async_context_wait_for_work_until(cyw43_arch_async_context(), next_sensor_read);
        #else
        // For regular Pico without BLE, just sleep until next sensor read
        sleep_until(next_sensor_read);
        #endif

        // Check if it's time to read sensor
        if (time_reached(next_sensor_read)) {
            // Toggle LED to show activity
            led_state = !led_state;
            led_set(led_state);

            // Read compensated sensor data
            rslt = bme280_get_sensor_data(BME280_ALL, &comp_data, &dev);
            
            if (rslt == BME280_OK) {
                // Convert sensor data to standard units
                float temperature, pressure, humidity;
                #ifdef BME280_DOUBLE_ENABLE
                temperature = comp_data.temperature;
                pressure = comp_data.pressure / 100.0f;  // Convert Pa to hPa
                humidity = comp_data.humidity;
                #else
                temperature = comp_data.temperature / 100.0f;
                pressure = comp_data.pressure / 100.0f;  // Convert Pa to hPa
                humidity = comp_data.humidity / 1024.0f;
                #endif
                
                // Print results
                printf("Temperature: %.2f C\n", temperature);
                printf("Pressure:    %.3f kPa\n", pressure);
                printf("Humidity:    %.2f %%\n", humidity);
                printf("---\n");
                
                // Update BLE advertisement with sensor data (only on Pico W)
                #ifdef CYW43_WL_GPIO_LED_PIN
                if (ble_advertise_is_ready()) {
                    sensor_data_t sensor_data = {
                        .temperature = temperature,
                        .pressure = pressure,
                        .humidity = humidity
                    };
                    ble_advertise_update(&sensor_data);
                }
                #endif
            } else {
                printf("Failed to read sensor data. Error code: %d\n", rslt);
            }

            // Schedule next sensor read (1 second from now)
            next_sensor_read = make_timeout_time_ms(POLL_INTERVAL_MS);
        }
    }
#endif
}
