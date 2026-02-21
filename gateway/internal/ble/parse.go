package ble

import (
	"encoding/binary"
	"fmt"
)

// Sensor payload format (little-endian): magic 0x01 0xD0, device_id uint8,
// run_count16 uint16, seq24 uint24, temperature int24 (scaled by 100), pressure uint24 (scaled by 10),
// humidity uint24 (scaled by 100), battery uint16 (scaled by 1000), battery_soc uint16 (scaled by 100, 0xFFFF = invalid) (21 bytes total).
const (
	sensorPayloadMagic0     = 0x01
	sensorPayloadMagic1     = 0xD0
	sensorPayloadLen        = 21
	sensorPayloadSOCInvalid = 0xFFFF
)

// SensorReading is a parsed BLE sensor advertisement (device_id + T/P/H/V/SOC + reading_id for dedup).
// DeviceID is the 8-bit advertised value (0–255). BatterySOC is 0–100 if valid; -1 if not available (0xFFFF or missing).
type SensorReading struct {
	DeviceID    uint32
	RunCount    uint32
	ReadingID   uint32
	Temperature float64
	Pressure    float64
	Humidity    float64
	Battery     float64
	BatterySOC  float64 // 0–100 or -1 if invalid/unknown
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
	deviceID := uint32(data[2])
	runCount := uint32(binary.LittleEndian.Uint16(data[3:5]))
	// Reading ID: uint24 (little-endian, 3 bytes)
	readingID := uint32(data[5]) | uint32(data[6])<<8 | uint32(data[7])<<16
	// Temperature: int24 LE scaled by 100 (0.01°C resolution)
	tempU := uint32(data[8]) | uint32(data[9])<<8 | uint32(data[10])<<16
	tempScaled := int32(tempU)
	if tempU&0x800000 != 0 {
		tempScaled |= ^int32(0xFFFFFF)
	}
	temp := float64(tempScaled) / 100.0
	// Pressure: uint24 LE scaled by 10 (0.1 kPa resolution)
	pressScaled := uint32(data[11]) | uint32(data[12])<<8 | uint32(data[13])<<16
	press := float64(pressScaled) / 10.0
	// Humidity: uint24 LE scaled by 100 (0.01% resolution)
	humScaled := uint32(data[14]) | uint32(data[15])<<8 | uint32(data[16])<<16
	hum := float64(humScaled) / 100.0
	// Battery voltage: uint16 LE scaled by 1000 (0.001V resolution)
	batteryScaled := binary.LittleEndian.Uint16(data[17:19])
	battery := float64(batteryScaled) / 1000.0
	// Battery SOC: uint16 LE scaled by 100 (0.01% resolution); 0xFFFF = invalid
	socScaled := binary.LittleEndian.Uint16(data[19:21])
	batterySOC := float64(-1)
	if socScaled != sensorPayloadSOCInvalid && socScaled <= 10000 {
		batterySOC = float64(socScaled) / 100.0
	}
	return &SensorReading{
		DeviceID:    deviceID,
		RunCount:    runCount,
		ReadingID:   readingID,
		Temperature: temp,
		Pressure:    press,
		Humidity:    hum,
		Battery:     battery,
		BatterySOC:  batterySOC,
	}, nil
}
