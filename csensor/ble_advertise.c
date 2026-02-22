/**
 * BLE Advertising Implementation for CloudPico
 * 
 * Uses btstack to advertise BME280 sensor data via BLE manufacturer data.
 */

#include <stdio.h>
#include <string.h>
#include <math.h>
#include <limits.h>
#include "ble_advertise.h"
#include "btstack.h"
#include "pico/cyw43_arch.h"
#include "pico/btstack_cyw43.h" 
#include "pico/stdlib.h"
#include "pico/time.h"
#include "pico/async_context.h"

#define ADV_INTERVAL_MIN_MS 800
#define ADV_INTERVAL_MAX_MS 800
#define ADV_TYPE 0  // ADV_IND: connectable undirected advertising

// BLE state
static bool ble_initialized = false;
static bool btstack_setup = false;  // Track if BTstack has been set up (one-time init)
static bool advertising_enabled = false;  // Track if advertising is actually enabled
static uint32_t device_id = 0;
static uint32_t reading_id = 0;
static btstack_packet_callback_registration_t hci_event_callback_registration;

// Advertisement data buffer (max 31 bytes for BLE)
// Format: Flags (3) + Manufacturer Data (28 bytes: 1 length + 1 type + 2 company ID + 24 payload) = 31 bytes
static uint8_t adv_data[31];
static uint8_t adv_data_len = 0;

/* int24 range for temperature: -8388608 .. 8388607 (scaled by 100) */
#define TEMP_SCALED_MIN  (-8388608L)
#define TEMP_SCALED_MAX   (8388607L)
#define UINT24_MAX       (16777215UL)

/**
 * Build manufacturer data payload
 * Format: magic (2) + device_id (1) + run_count16 (2) + seq24 (3) + temp (3) + pressure (3) + humidity (3) + battery (2) + soc (2) = 21 bytes
 *
 * device_id: lower 8 bits of uint32_t (0–255).
 * Temperature: int24 LE scaled by 100 (0.01°C resolution).
 * Pressure: uint24 LE scaled by 10 (0.1 kPa resolution).
 * Humidity: uint24 LE scaled by 100 (0.01% resolution).
 * Voltage: uint16 LE scaled by 1000 (0.001V resolution).
 * SOC: uint16 LE scaled by 100 (0.01% resolution), 0xFFFF = invalid/unknown.
 */
static void build_manufacturer_data(uint8_t *buffer, uint32_t dev_id, uint32_t run_count, uint32_t read_id,
                                    float temp, float pressure, float humidity, float battery, float soc) {
    uint8_t *p = buffer;

    // Magic bytes
    *p++ = BLE_MAGIC_0;
    *p++ = BLE_MAGIC_1;

    // Device ID (1 byte) - extract lower 8 bits from uint32_t
    *p++ = (uint8_t)(dev_id & 0xFF);

    // Run count (little-endian uint16)
    uint16_t run_count_16 = (uint16_t)(run_count & 0xFFFF);
    *p++ = (uint8_t)(run_count_16 & 0xFF);
    *p++ = (uint8_t)((run_count_16 >> 8) & 0xFF);

    // Reading ID / Sequence (little-endian uint24)
    uint32_t read_id_24 = read_id & 0xFFFFFF;
    *p++ = (uint8_t)(read_id_24 & 0xFF);
    *p++ = (uint8_t)((read_id_24 >> 8) & 0xFF);
    *p++ = (uint8_t)((read_id_24 >> 16) & 0xFF);

    // Temperature (little-endian int24, scaled by 100: 0.01°C resolution)
    long temp_rounded = isfinite(temp) ? lrintf(temp * 100.0f) : 0L;
    if (temp_rounded < TEMP_SCALED_MIN) temp_rounded = TEMP_SCALED_MIN;
    if (temp_rounded > TEMP_SCALED_MAX) temp_rounded = TEMP_SCALED_MAX;
    uint32_t temp_u = (uint32_t)(int32_t)temp_rounded;
    *p++ = (uint8_t)(temp_u & 0xFF);
    *p++ = (uint8_t)((temp_u >> 8) & 0xFF);
    *p++ = (uint8_t)((temp_u >> 16) & 0xFF);

    // Pressure (little-endian uint24, scaled by 10: 0.1 kPa resolution)
    long press_rounded = (isfinite(pressure) && pressure >= 0.0f) ? lrintf(pressure * 10.0f) : 0L;
    if (press_rounded < 0) press_rounded = 0;
    if ((unsigned long)press_rounded > UINT24_MAX) press_rounded = (long)UINT24_MAX;
    uint32_t press_u = (uint32_t)press_rounded;
    *p++ = (uint8_t)(press_u & 0xFF);
    *p++ = (uint8_t)((press_u >> 8) & 0xFF);
    *p++ = (uint8_t)((press_u >> 16) & 0xFF);

    // Humidity (little-endian uint24, scaled by 100: 0.01% resolution)
    long hum_rounded = (isfinite(humidity) && humidity >= 0.0f) ? lrintf(humidity * 100.0f) : 0L;
    if (hum_rounded < 0) hum_rounded = 0;
    if ((unsigned long)hum_rounded > UINT24_MAX) hum_rounded = (long)UINT24_MAX;
    uint32_t hum_u = (uint32_t)hum_rounded;
    *p++ = (uint8_t)(hum_u & 0xFF);
    *p++ = (uint8_t)((hum_u >> 8) & 0xFF);
    *p++ = (uint8_t)((hum_u >> 16) & 0xFF);

    // Battery voltage (little-endian uint16, scaled by 1000: 0.001V resolution)
    long battery_rounded = (isfinite(battery) && battery >= 0.0f) ? lrintf(battery * 1000.0f) : 0L;
    if (battery_rounded < 0) battery_rounded = 0;
    if ((unsigned long)battery_rounded > UINT16_MAX) battery_rounded = UINT16_MAX;
    uint16_t battery_scaled = (uint16_t)battery_rounded;
    *p++ = (uint8_t)(battery_scaled & 0xFF);
    *p++ = (uint8_t)((battery_scaled >> 8) & 0xFF);

    // Battery SOC (little-endian uint16, scaled by 100: 0.01% resolution; 0xFFFF = invalid)
    uint16_t soc_scaled = 0xFFFF;
    if (isfinite(soc) && soc >= 0.0f && soc <= 100.0f) {
        long s = lrintf(soc * 100.0f);
        if (s < 0) s = 0;
        if (s > 10000) s = 10000;
        soc_scaled = (uint16_t)s;
    }
    *p++ = (uint8_t)(soc_scaled & 0xFF);
    *p++ = (uint8_t)((soc_scaled >> 8) & 0xFF);
}

/**
 * Build advertisement data packet
 */
static void build_adv_data(uint32_t dev_id, uint32_t run_count, uint32_t read_id,
                          float temp, float pressure, float humidity, float battery, float soc) {
    uint8_t *p = adv_data;

    // Flags: general discoverable, BR/EDR not supported
    *p++ = 0x02;  // Length
    *p++ = BLUETOOTH_DATA_TYPE_FLAGS;
    *p++ = 0x06;  // Flags value

    // Manufacturer data: Company ID (2 bytes) + payload (21 bytes) = 23 bytes total
    uint8_t mfg_data[21];
    build_manufacturer_data(mfg_data, dev_id, run_count, read_id, temp, pressure, humidity, battery, soc);

    *p++ = 24;  // Length: 1 (type) + 2 (Company ID) + 21 (payload)
    *p++ = BLUETOOTH_DATA_TYPE_MANUFACTURER_SPECIFIC_DATA;
    // Company ID (little-endian)
    *p++ = (uint8_t)(BLE_COMPANY_ID & 0xFF);
    *p++ = (uint8_t)((BLE_COMPANY_ID >> 8) & 0xFF);
    // Payload
    memcpy(p, mfg_data, 21);
    p += 21;
    
    adv_data_len = p - adv_data;
    
    // BLE limitation: max 31 bytes
    if (adv_data_len > 31) {
        printf("ERROR: Advertisement data too long: %d bytes\n", adv_data_len);
        adv_data_len = 31;
    }
}

/**
 * Packet handler for BLE events
 */
static void packet_handler(uint8_t packet_type, uint16_t channel, uint8_t *packet, uint16_t size) {
    UNUSED(size);
    UNUSED(channel);
    
    if (packet_type != HCI_EVENT_PACKET) return;
    
    uint8_t event_type = hci_event_packet_get_type(packet);
    switch(event_type) {
        case BTSTACK_EVENT_STATE:
            if (btstack_event_state_get_state(packet) != HCI_STATE_WORKING) return;
            
            bd_addr_t local_addr;
            gap_local_bd_addr(local_addr);
            printf("BLE: BTstack up and running on %s\n", bd_addr_to_str(local_addr));
            
            // Setup advertisement parameters
            bd_addr_t null_addr;
            memset(null_addr, 0, 6);
            gap_advertisements_set_params(ADV_INTERVAL_MIN_MS, ADV_INTERVAL_MAX_MS, 
                                          ADV_TYPE, 0, null_addr, 0x07, 0x00);
            
            // Set initial advertisement data (will be updated with sensor data before enabling)
            build_adv_data(device_id, 0, reading_id, 0.0f, 0.0f, 0.0f, 0.0f, -1.0f);
            gap_advertisements_set_data(adv_data_len, adv_data);
            // Don't enable advertising yet - wait for sensor data update
            // gap_advertisements_enable(1);
            
            printf("BLE: BTstack ready (device_id: 0x%08lX, advertised as 0x%04X), waiting for sensor data...\n", 
                   (unsigned long)device_id, (uint16_t)(device_id & 0xFFFF));
            ble_initialized = true;
            advertising_enabled = false;
            break;
            
        default:
            break;
    }
}

int ble_advertise_init(uint32_t dev_id) {
    if (ble_initialized) {
        printf("BLE: Already initialized\n");
        return 0;
    }
    
    device_id = dev_id;
    
    // One-time setup of BTstack (only do this once)
    if (!btstack_setup) {
        // Note: cyw43_arch_init() may have already been called by pico_led_init()
        // For Pico W, cyw43_arch_init() can be called multiple times safely
        // It returns 0 on success, or non-zero if already initialized or on error
        int init_result = cyw43_arch_init();
        if (init_result != 0) {
            // If it's already initialized, that's fine - continue
            // But if it's a real error, we should fail
            printf("BLE: cyw43_arch_init returned %d (may be already initialized)\n", init_result);
            // Continue anyway - if cyw43 is already initialized, we can still use it
        }
        
        // Initialize BTstack (one-time setup)
        l2cap_init();
        sm_init();
        
        // Register packet handler (one-time setup)
        hci_event_callback_registration.callback = &packet_handler;
        hci_add_event_handler(&hci_event_callback_registration);
        
        btstack_setup = true;
        // Only reset reading_id on first initialization (device boot)
        reading_id = 0;
        printf("BLE: BTstack setup complete (reading_id reset to 0)\n");
    }
    // Don't reset reading_id on subsequent initializations - keep incrementing across sessions
    
    // Turn on Bluetooth power
    hci_power_control(HCI_POWER_ON);
    
    printf("BLE: Power on, waiting for BTstack to be ready...\n");
    return 0;
}

/**
 * Process CYW43/BLE async work until the given deadline.
 * Polls once to handle any pending work, then blocks until either new work
 * arrives or the absolute time `until` is reached. Use in the main loop to
 * keep the BLE stack running while waiting (e.g. between ad updates).
 */
void ble_advertise_poll_until(absolute_time_t until) {
    async_context_poll(cyw43_arch_async_context());
    async_context_wait_for_work_until(cyw43_arch_async_context(), until);
}

int ble_advertise_start(uint32_t run_count, sensor_data_t *data) {
    if (!ble_initialized) {
        return -1;
    }
    
    if (data == NULL) {
        return -1;
    }
    
    // Increment reading ID for each update
    reading_id++;
    
    // Build new advertisement data
    build_adv_data(device_id, run_count, reading_id, data->temperature, data->pressure, data->humidity, data->battery, data->battery_soc);
    
    // Update advertisement data
    gap_advertisements_set_data(adv_data_len, adv_data);
    
    // Enable advertising if not already enabled (first update with real sensor data)
    if (!advertising_enabled) {
        gap_advertisements_enable(1);
        advertising_enabled = true;
        printf("BLE: Advertising enabled with sensor data (reading_id: %lu, T: %.2f°C, P: %.2f kPa, H: %.2f%%, V: %.3fV, SOC: %.0f%%)\n",
               reading_id, data->temperature, data->pressure, data->humidity, data->battery, data->battery_soc >= 0.0f ? data->battery_soc : -1.0f);
    } else {
        printf("BLE: Updated advertisement (reading_id: %lu, T: %.2f°C, P: %.2f kPa, H: %.2f%%, V: %.3fV, SOC: %.0f%%)\n",
               reading_id, data->temperature, data->pressure, data->humidity, data->battery, data->battery_soc >= 0.0f ? data->battery_soc : -1.0f);
    }
    
    return 0;
}

void ble_advertise_stop(void) {
    if (!advertising_enabled) {
        return;
    }
    gap_advertisements_enable(0);
    advertising_enabled = false;
}

void ble_advertise_deinit(void) {
    if (!ble_initialized) {
        return;
    }
    ble_advertise_stop();
    // Turn off Bluetooth power (but keep BTstack setup intact)
    hci_power_control(HCI_POWER_OFF);
    
    // Reset the initialized flag (BTstack setup remains)
    ble_initialized = false;
    printf("BLE: Advertising disabled, power off\n");
}

bool ble_advertise_is_ready(void) {
    return ble_initialized;
}
