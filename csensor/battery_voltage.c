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
    unsigned long sum = 0;  /* unsigned long for linter compatibility */
    for (int i = 0; i < BATTERY_ADC_SAMPLES; i++) {
        sum += adc_read();
        sleep_us(BATTERY_ADC_SAMPLE_DELAY_US);
    }
    float raw = (float)sum / (float)BATTERY_ADC_SAMPLES;

    float v_adc = raw * BATTERY_ADC_REF_V / (float)BATTERY_ADC_MAX_COUNT;
    float v_bat = v_adc * BATTERY_DIVIDER_RATIO;
    return v_bat;
}
