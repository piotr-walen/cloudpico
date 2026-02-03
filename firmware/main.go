package main

import (
	"fmt"
	"machine"
	"time"

	"tinygo.org/x/drivers/bme280"
)

func main() {
	// USB serial
	machine.Serial.Configure(machine.UARTConfig{})

	// I2C0 on GP0/GP1
	i2c := machine.I2C0
	if err := i2c.Configure(machine.I2CConfig{
		SDA:       machine.GP0,
		SCL:       machine.GP1,
		Frequency: 400 * machine.KHz,
	}); err != nil {
		fmt.Printf("I2C configure error: %v\r\n", err)
		for {
			time.Sleep(time.Second)
		}
	}

	sensor := bme280.New(i2c)

	sensor.Configure()
	fmt.Printf("BME280 init OK\r\n")

	for {
		t, errT := sensor.ReadTemperature()
		p, errP := sensor.ReadPressure()
		h, errH := sensor.ReadHumidity()

		if errT != nil || errP != nil || errH != nil {
			fmt.Printf("read error: T=%v P=%v H=%v\r\n", errT, errP, errH)
		} else {
			// Driver units:
			// t = milli-°C, p = milli-Pa, h = hundredths of %RH
			tempC := float32(t) / 1000.0
			pressHPa := float32(p) / 100000.0
			humPct := float32(h) / 100.0

			fmt.Printf("T: %.2f C  P: %.2f hPa  H: %.2f %%\r\n", tempC, pressHPa, humPct)
		}

		time.Sleep(2 * time.Second)
	}
}
