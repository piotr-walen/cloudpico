// BME280 I2C sensor reading (temperature, pressure, humidity).
package main

import (
	"machine"

	"tinygo.org/x/drivers/bme280"
)

// RunSensor configures I2C and BME280, then blocks in a loop reading and
// printing T/P/H every 2 seconds.

type Reading struct {
	Temperature float32
	Pressure    float32
	Humidity    float32
}

type Sensor struct {
	device *bme280.Device
}

func NewSensor() (Sensor, error) {
	i2c := machine.I2C1
	if err := i2c.Configure(machine.I2CConfig{
		SDA:       machine.GP32,
		SCL:       machine.GP33,
		Frequency: 400 * machine.KHz,
	}); err != nil {
		return Sensor{}, err
	}

	sensor := bme280.New(i2c)
	sensor.Configure()

	return Sensor{
		device: &sensor,
	}, nil
}

func (s *Sensor) Read() (Reading, error) {

	t, errT := s.device.ReadTemperature()
	if errT != nil {
		return Reading{}, errT
	}
	p, errP := s.device.ReadPressure()
	if errP != nil {
		return Reading{}, errP
	}
	h, errH := s.device.ReadHumidity()
	if errH != nil {
		return Reading{}, errH
	}

	tempC := float32(t) / 1000.0
	pressHPa := float32(p) / 100000.0
	humPct := float32(h) / 100.0
	return Reading{
		Temperature: tempC,
		Pressure:    pressHPa,
		Humidity:    humPct,
	}, nil

}
