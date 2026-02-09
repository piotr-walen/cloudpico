// BLE advertising for Pico 2 W so the gateway can discover the device.
// Manufacturer data format: [0:2] magic 0x01 0xD0, [2:6] reading_id uint32 LE,
// [6:10] temp float32 LE, [10:14] pressure float32 LE, [14:18] humidity float32 LE.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	blePayloadMagic0 = 0x01
	blePayloadMagic1 = 0xD0
	blePayloadMinLen = 18
)

var adapter = bluetooth.DefaultAdapter

type SendAdvertisementsOptions struct {
	Interval time.Duration
	Duration time.Duration
}

// EncodeReadingPayload builds the manufacturer data payload: magic (2) + reading_id (4) + T/P/H (12).
func EncodeReadingPayload(reading Reading) ([]byte, uint32) {
	id := rand.Uint32()
	buf := make([]byte, blePayloadMinLen)
	buf[0] = blePayloadMagic0
	buf[1] = blePayloadMagic1
	binary.LittleEndian.PutUint32(buf[2:6], id)
	binary.LittleEndian.PutUint32(buf[6:10], math.Float32bits(reading.Temperature))
	binary.LittleEndian.PutUint32(buf[10:14], math.Float32bits(reading.Pressure))
	binary.LittleEndian.PutUint32(buf[14:18], math.Float32bits(reading.Humidity))
	return buf, id
}

func SendAdvertisements(sensorReading Reading, options SendAdvertisementsOptions) error {
	if options.Interval == 0 {
		options.Interval = 100 * time.Millisecond
	}
	if options.Duration == 0 {
		options.Duration = 10 * time.Second
	}

	fmt.Println("ble: enabling adapter...")
	if err := adapter.Enable(); err != nil {
		fmt.Println("FATAL: adapter.Enable failed:", err)
		return err
	}
	fmt.Println("ble: adapter enabled")

	payload, id := EncodeReadingPayload(sensorReading)
	fmt.Printf("ble: payload id=%d T=%.2f P=%.2f H=%.2f\r\n", id, sensorReading.Temperature, sensorReading.Pressure, sensorReading.Humidity)

	adv := adapter.DefaultAdvertisement()
	fmt.Println("ble: configuring advertisement...")
	if err := adv.Configure(bluetooth.AdvertisementOptions{
		AdvertisementType: bluetooth.AdvertisingTypeNonConnInd,
		LocalName:         "pico2w-sensor",
		Interval:          bluetooth.NewDuration(options.Interval),
		ManufacturerData: []bluetooth.ManufacturerDataElement{
			{CompanyID: 0xFFFF, Data: payload},
		},
	}); err != nil {
		fmt.Println("FATAL: adv.Configure failed:", err)
		return err
	}
	fmt.Println("ble: advertisement configured")

	adv.Start()
	time.AfterFunc(options.Duration, func() {
		adv.Stop()
	})
	return nil
}
