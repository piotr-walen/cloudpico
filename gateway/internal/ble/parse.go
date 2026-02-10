package ble

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Sensor payload format (little-endian): magic 0x01 0xD0, reading_id uint32,
// temperature float32, pressure float32, humidity float32 (18 bytes total).
const (
	sensorPayloadMagic0 = 0x01
	sensorPayloadMagic1 = 0xD0
	sensorPayloadLen   = 18
)

// SensorReading is a parsed BLE sensor advertisement (T/P/H + id for dedup).
type SensorReading struct {
	ReadingID   uint32
	Temperature float64
	Pressure    float64
	Humidity    float64
}

// ParseSensorPayload parses manufacturer data from a Pico sensor advertisement.
// Returns (nil, error) if the payload is not the expected format or length.
func ParseSensorPayload(data []byte) (*SensorReading, error) {
	if len(data) < sensorPayloadLen {
		return nil, fmt.Errorf("payload too short: %d", len(data))
	}
	if data[0] != sensorPayloadMagic0 || data[1] != sensorPayloadMagic1 {
		return nil, fmt.Errorf("invalid magic: %02X %02X", data[0], data[1])
	}
	id := binary.LittleEndian.Uint32(data[2:6])
	temp := math.Float32frombits(binary.LittleEndian.Uint32(data[6:10]))
	press := math.Float32frombits(binary.LittleEndian.Uint32(data[10:14]))
	hum := math.Float32frombits(binary.LittleEndian.Uint32(data[14:18]))
	return &SensorReading{
		ReadingID:   id,
		Temperature: float64(temp),
		Pressure:    float64(press),
		Humidity:    float64(hum),
	}, nil
}
