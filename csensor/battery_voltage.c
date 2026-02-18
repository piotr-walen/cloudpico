#include <stdio.h>
#include <stdint.h>
#include "pico/stdlib.h"
#include "hardware/adc.h"
#include "battery_voltage.h"

void battery_voltage_init(void) {
    // Init ADC and set up GPIO26 as ADC0
    adc_init();
    adc_gpio_init(26);     // GPIO26 = ADC0
    adc_select_input(0);   // select ADC0
}

float read_battery_voltage(void) {
    // ADC on Pico is 12-bit: 0..4095 for 0..3.3V (approx, referenced to 3V3)
    const float adc_ref = 3.3f;
    const float divider_ratio = 2.0f;  // 100k/100k divider => battery = Vadc * 2

    // Simple averaging to reduce noise
    const int samples = 32;
    unsigned long sum = 0;  // Using unsigned long for linter compatibility
    for (int i = 0; i < samples; i++) {
        sum += adc_read();
        sleep_us(200);
    }
    float raw = (float)sum / (float)samples;

    float v_adc = raw * adc_ref / 4095.0f;
    float v_bat = v_adc * divider_ratio;
    return v_bat;
}
