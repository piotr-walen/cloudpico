package ble

import (
	"encoding/binary"
	"fmt"
)

// Sensor payload format (little-endian): magic 0x01 0xD0, device_id uint16,
// run_count16 uint16, seq24 uint24, temperature int16 (scaled by 100), pressure uint16 (scaled by 10),
// humidity uint16 (scaled by 100), battery uint16 (scaled by 1000) (17 bytes total).
const (
	sensorPayloadMagic0 = 0x01
	sensorPayloadMagic1 = 0xD0
	sensorPayloadLen    = 17
)

// SensorReading is a parsed BLE sensor advertisement (device_id + T/P/H/V + reading_id for dedup).
type SensorReading struct {
	DeviceID    uint32
	RunCount    uint32
	ReadingID   uint32
	Temperature float64
	Pressure    float64
	Humidity    float64
	Battery     float64
}

// ParseSensorPayload parses manufacturer data from a Pico sensor advertisement.
// Returns (nil, error) if the payload is not the expected format or length.
func ParseSensorPayload(data []byte) (*SensorReading, error) {
	if len(data) < sensorPayloadLen {
		return nil, fmt.Errorf("payload too short: got %d bytes, expected %d", len(data), sensorPayloadLen)
	}
	if data[0] != sensorPayloadMagic0 || data[1] != sensorPayloadMagic1 {
		return nil, fmt.Errorf("invalid magic: got %02X %02X, expected %02X %02X",
			data[0], data[1], sensorPayloadMagic0, sensorPayloadMagic1)
	}
	deviceID := uint32(binary.LittleEndian.Uint16(data[2:4]))
	runCount := uint32(binary.LittleEndian.Uint16(data[4:6]))
	// Reading ID: uint24 (little-endian, 3 bytes)
	readingID := uint32(data[6]) | uint32(data[7])<<8 | uint32(data[8])<<16
	// Temperature: int16 scaled by 100 (0.01°C resolution)
	tempScaled := int16(binary.LittleEndian.Uint16(data[9:11]))
	temp := float64(tempScaled) / 100.0
	// Pressure: uint16 scaled by 10 (0.1 kPa resolution)
	pressScaled := binary.LittleEndian.Uint16(data[11:13])
	press := float64(pressScaled) / 10.0
	// Humidity: uint16 scaled by 100 (0.01% resolution)
	humScaled := binary.LittleEndian.Uint16(data[13:15])
	hum := float64(humScaled) / 100.0
	// Battery voltage: uint16 scaled by 1000 (0.001V resolution)
	batteryScaled := binary.LittleEndian.Uint16(data[15:17])
	battery := float64(batteryScaled) / 1000.0
	return &SensorReading{
		DeviceID:    deviceID,
		RunCount:    runCount,
		ReadingID:   readingID,
		Temperature: temp,
		Pressure:    press,
		Humidity:    hum,
		Battery:     battery,
	}, nil
}
