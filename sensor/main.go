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
	"time"
)

func main() {
	machine.Serial.Configure(machine.UARTConfig{})
	time.Sleep(1500 * time.Millisecond)
	fmt.Println("boot: pico2w BLE beacon + BME280 sensor")

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		// Read sensor values
		reading, err := ReadSensorValues()
		if err != nil {
			fmt.Printf("ERROR: sensor read failed: %v\r\n", err)
			continue
		}

		// Print sensor values
		fmt.Printf("T: %.2f C  P: %.2f hPa  H: %.2f %%\r\n", reading.Temperature, reading.Pressure, reading.Humidity)

		// Update BLE advertisement with new sensor data
		if err := SendAdvertisements(reading, SendAdvertisementsOptions{}); err != nil {
			fmt.Printf("ERROR: BLE advertisement update failed: %v\r\n", err)
			continue
		}
		fmt.Println("ble: advertisement updated with new sensor data")
	}
}
