// BLE advertising for Pico 2 W so the gateway can discover the device.
// Manufacturer data format: [0:2] magic 0x01 0xD0, [2:6] device_id uint32 LE,
// [6:10] reading_id uint32 LE, [10:14] temp float32 LE, [14:18] pressure float32 LE,
// [18:22] humidity float32 LE (22 bytes total).
package main

import (
	"encoding/binary"
	"math"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	blePayloadMagic0 = 0x01
	blePayloadMagic1 = 0xD0
	blePayloadMinLen = 22
)

type SendAdvertisementsOptions struct {
	Interval time.Duration
	Duration time.Duration
}

type BLE struct {
	deviceID             uint32
	adapter              *bluetooth.Adapter
	readingData          [blePayloadMinLen]byte
	advertisementOptions bluetooth.AdvertisementOptions
	advertisement        bluetooth.Advertisement

	sleepDuration time.Duration
}

func NewBLE(deviceID uint32, options SendAdvertisementsOptions) (*BLE, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, err
	}

	ble := &BLE{
		adapter:       adapter,
		deviceID:      deviceID,
		readingData:   [blePayloadMinLen]byte{},
		advertisement: *adapter.DefaultAdvertisement(),
		sleepDuration: options.Duration,
	}
	ble.advertisementOptions = bluetooth.AdvertisementOptions{
		AdvertisementType: bluetooth.AdvertisingTypeNonConnInd,
		LocalName:         "pico2w-sensor",
		Interval:          bluetooth.NewDuration(options.Interval),
		ManufacturerData: []bluetooth.ManufacturerDataElement{
			{CompanyID: 0xFFFF, Data: ble.readingData[:]},
		},
	}
	return ble, nil
}

var counter uint32 = 0

// EncodeReadingPayload builds the manufacturer data payload: magic (2) + device_id (4) + reading_id (4) + T/P/H (12).
// Uses the reusable payloadBuf to avoid heap allocations.
func (b *BLE) EncodeReadingPayload(reading Reading, id uint32) {

	b.readingData[0] = blePayloadMagic0
	b.readingData[1] = blePayloadMagic1
	binary.LittleEndian.PutUint32(b.readingData[2:6], b.deviceID)
	binary.LittleEndian.PutUint32(b.readingData[6:10], id)
	binary.LittleEndian.PutUint32(b.readingData[10:14], math.Float32bits(reading.Temperature))
	binary.LittleEndian.PutUint32(b.readingData[14:18], math.Float32bits(reading.Pressure))
	binary.LittleEndian.PutUint32(b.readingData[18:22], math.Float32bits(reading.Humidity))
}

func (b *BLE) Send(sensorReading Reading) (uint32, error) {
	id := counter
	counter++

	b.EncodeReadingPayload(sensorReading, id)

	if err := b.advertisement.Configure(b.advertisementOptions); err != nil {
		return 0, err
	}

	if err := b.advertisement.Start(); err != nil {
		b.advertisement.Stop()
		return 0, err
	}

	time.Sleep(b.sleepDuration)
	b.advertisement.Stop()
	return id, nil
}
