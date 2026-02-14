/**
 * BLE Advertising Implementation for CloudPico
 * 
 * Uses btstack to advertise BME280 sensor data via BLE manufacturer data.
 */

#include <stdio.h>
#include <string.h>
#include <math.h>
#include "ble_advertise.h"
#include "btstack.h"
#include "pico/cyw43_arch.h"
#include "pico/btstack_cyw43.h"
#include "pico/stdlib.h"

#define ADV_INTERVAL_MIN_MS 800
#define ADV_INTERVAL_MAX_MS 800
#define ADV_TYPE 0  // ADV_IND: connectable undirected advertising

// BLE state
static bool ble_initialized = false;
static uint32_t device_id = 0;
static uint32_t reading_id = 0;
static btstack_packet_callback_registration_t hci_event_callback_registration;

// Advertisement data buffer (max 31 bytes for BLE)
// Format: Flags (3) + Name (up to 20) + Manufacturer Data (25)
static uint8_t adv_data[31];
static uint8_t adv_data_len = 0;

/**
 * Build manufacturer data payload
 * Format: magic (2) + device_id (4) + reading_id (4) + temp (4) + pressure (4) + humidity (4) = 22 bytes
 */
static void build_manufacturer_data(uint8_t *buffer, uint32_t dev_id, uint32_t read_id, 
                                     float temp, float pressure, float humidity) {
    uint8_t *p = buffer;
    
    // Magic bytes
    *p++ = BLE_MAGIC_0;
    *p++ = BLE_MAGIC_1;
    
    // Device ID (little-endian uint32)
    *p++ = (uint8_t)(dev_id & 0xFF);
    *p++ = (uint8_t)((dev_id >> 8) & 0xFF);
    *p++ = (uint8_t)((dev_id >> 16) & 0xFF);
    *p++ = (uint8_t)((dev_id >> 24) & 0xFF);
    
    // Reading ID (little-endian uint32)
    *p++ = (uint8_t)(read_id & 0xFF);
    *p++ = (uint8_t)((read_id >> 8) & 0xFF);
    *p++ = (uint8_t)((read_id >> 16) & 0xFF);
    *p++ = (uint8_t)((read_id >> 24) & 0xFF);
    
    // Temperature (little-endian float32)
    uint32_t temp_bits = *(uint32_t*)&temp;
    *p++ = (uint8_t)(temp_bits & 0xFF);
    *p++ = (uint8_t)((temp_bits >> 8) & 0xFF);
    *p++ = (uint8_t)((temp_bits >> 16) & 0xFF);
    *p++ = (uint8_t)((temp_bits >> 24) & 0xFF);
    
    // Pressure (little-endian float32)
    uint32_t press_bits = *(uint32_t*)&pressure;
    *p++ = (uint8_t)(press_bits & 0xFF);
    *p++ = (uint8_t)((press_bits >> 8) & 0xFF);
    *p++ = (uint8_t)((press_bits >> 16) & 0xFF);
    *p++ = (uint8_t)((press_bits >> 24) & 0xFF);
    
    // Humidity (little-endian float32)
    uint32_t hum_bits = *(uint32_t*)&humidity;
    *p++ = (uint8_t)(hum_bits & 0xFF);
    *p++ = (uint8_t)((hum_bits >> 8) & 0xFF);
    *p++ = (uint8_t)((hum_bits >> 16) & 0xFF);
    *p++ = (uint8_t)((hum_bits >> 24) & 0xFF);
}

/**
 * Build advertisement data packet
 */
static void build_adv_data(uint32_t dev_id, uint32_t read_id, 
                           float temp, float pressure, float humidity) {
    uint8_t *p = adv_data;
    
    // Flags: general discoverable, BR/EDR not supported
    *p++ = 0x02;  // Length
    *p++ = BLUETOOTH_DATA_TYPE_FLAGS;
    *p++ = 0x06;  // Flags value
    
    // Manufacturer data: Company ID (2 bytes) + payload (22 bytes) = 24 bytes total
    uint8_t mfg_data[22];
    build_manufacturer_data(mfg_data, dev_id, read_id, temp, pressure, humidity);
    
    *p++ = 25;  // Length: 1 (type) + 2 (Company ID) + 22 (payload)
    *p++ = BLUETOOTH_DATA_TYPE_MANUFACTURER_SPECIFIC_DATA;
    // Company ID (little-endian)
    *p++ = (uint8_t)(BLE_COMPANY_ID & 0xFF);
    *p++ = (uint8_t)((BLE_COMPANY_ID >> 8) & 0xFF);
    // Payload
    memcpy(p, mfg_data, 22);
    p += 22;
    
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
            
            // Set initial advertisement data (will be updated with sensor data)
            build_adv_data(device_id, reading_id, 0.0f, 0.0f, 0.0f);
            gap_advertisements_set_data(adv_data_len, adv_data);
            gap_advertisements_enable(1);
            
            printf("BLE: Advertising enabled (device_id: 0x%08X)\n", device_id);
            ble_initialized = true;
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
    reading_id = 0;
    
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
    
    // Initialize BTstack
    l2cap_init();
    sm_init();
    
    // Register packet handler
    hci_event_callback_registration.callback = &packet_handler;
    hci_add_event_handler(&hci_event_callback_registration);
    
    // Turn on Bluetooth
    hci_power_control(HCI_POWER_ON);
    
    printf("BLE: Initialization started (device_id: 0x%08X)\n", device_id);
    printf("BLE: Waiting for BTstack to be ready...\n");
    return 0;
}

int ble_advertise_update(sensor_data_t *data) {
    if (!ble_initialized) {
        return -1;
    }
    
    if (data == NULL) {
        return -1;
    }
    
    // Increment reading ID for each update
    reading_id++;
    
    // Build new advertisement data
    build_adv_data(device_id, reading_id, data->temperature, data->pressure, data->humidity);
    
    // Update advertisement
    gap_advertisements_set_data(adv_data_len, adv_data);
    
    printf("BLE: Updated advertisement (reading_id: %lu, T: %.2f°C, P: %.2f kPa, H: %.2f%%)\n",
           reading_id, data->temperature, data->pressure, data->humidity);
    
    return 0;
}

void ble_advertise_deinit(void) {
    if (!ble_initialized) {
        return;
    }
    
    gap_advertisements_enable(0);
    hci_power_control(HCI_POWER_OFF);
    cyw43_arch_deinit();
    
    ble_initialized = false;
    printf("BLE: Deinitialized\n");
}

bool ble_advertise_is_ready(void) {
    return ble_initialized;
}
