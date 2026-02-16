package ble

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Sensor payload format (little-endian): magic 0x01 0xD0, device_id uint16,
// run_count uint32, reading_id uint32, temperature float32, pressure float32, humidity float32 (24 bytes total).
const (
	sensorPayloadMagic0 = 0x01
	sensorPayloadMagic1 = 0xD0
	sensorPayloadLen    = 24
)

// SensorReading is a parsed BLE sensor advertisement (device_id + T/P/H + reading_id for dedup).
type SensorReading struct {
	DeviceID    uint32
	RunCount    uint32
	ReadingID   uint32
	Temperature float64
	Pressure    float64
	Humidity    float64
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
	runCount := binary.LittleEndian.Uint32(data[4:8])
	readingID := binary.LittleEndian.Uint32(data[8:12])
	temp := math.Float32frombits(binary.LittleEndian.Uint32(data[12:16]))
	press := math.Float32frombits(binary.LittleEndian.Uint32(data[16:20]))
	hum := math.Float32frombits(binary.LittleEndian.Uint32(data[20:24]))
	return &SensorReading{
		DeviceID:    deviceID,
		RunCount:    runCount,
		ReadingID:   readingID,
		Temperature: float64(temp),
		Pressure:    float64(press),
		Humidity:    float64(hum),
	}, nil
}
