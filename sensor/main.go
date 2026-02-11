// BLE beacon for Pico 2 W. Advertises continuously so the gateway can discover it.
// Also reads BME280 over I2C and prints T/P/H to serial.
//
// Build and flash (use pico2-w for the wireless board):
//
//	tinygo flash -target=pico2-w .
package main

import (
	"fmt"
	"machine"
	"strconv"
	"time"
)

const SENSOR_POLL_INTERVAL = 2000 * time.Millisecond
const BLE_ADVERTISEMENT_INTERVAL = 100 * time.Millisecond
const BLE_ADVERTISEMENT_DURATION = 420 * time.Millisecond
const BOOT_DELAY = 5000 * time.Millisecond

// deviceIDStr is set at build time via -ldflags "-X main.deviceIDStr=0x12345678"
// Format: -ldflags "-X main.deviceIDStr=0x12345678" or "-X main.deviceIDStr=305419896"
var deviceIDStr string

// parseDeviceIDFromStr parses deviceIDStr and returns the uint32 value.
// Returns 0 if deviceIDStr is empty or invalid.
func parseDeviceIDFromStr(s string) uint32 {
	if s == "" {
		return 0
	}
	var parsed uint64
	var err error
	if len(s) > 2 && s[0:2] == "0x" {
		parsed, err = strconv.ParseUint(s[2:], 16, 32)
	} else {
		parsed, err = strconv.ParseUint(s, 10, 32)
	}
	if err != nil {
		return 0
	}
	return uint32(parsed)
}

func main() {
	machine.Serial.Configure(machine.UARTConfig{})
	deviceID := parseDeviceIDFromStr(deviceIDStr)
	fmt.Printf("boot: pico2w BLE beacon + BME280 sensor (device_id: 0x%08X)\r\n", deviceID)

	ble, err := NewBLE(deviceID, SendAdvertisementsOptions{
		Interval: BLE_ADVERTISEMENT_INTERVAL,
		Duration: BLE_ADVERTISEMENT_DURATION,
	})
	if err != nil {
		fmt.Printf("ERROR: BLE initialization failed: %v\r\n", err)
		return
	}

	sensor, err := NewSensor()
	if err != nil {
		fmt.Printf("ERROR: sensor initialization failed: %v\r\n", err)
		return
	}

	for {
		reading, err := sensor.Read()

		if err != nil {
			time.Sleep(SENSOR_POLL_INTERVAL)
			continue
		}

		fmt.Println("Sending BLE advertisement...")
		reading_id, err := ble.Send(reading)
		if err != nil {
			fmt.Printf("ERROR: BLE advertisement update failed: %v\r\n", err)
			time.Sleep(SENSOR_POLL_INTERVAL)
			continue
		}
		fmt.Printf("BLE advertisement sent (reading_id: %d)\r\n", reading_id)

		time.Sleep(SENSOR_POLL_INTERVAL)
	}
}
