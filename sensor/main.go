// pico2w_adv_loop_serial_fmt/main.go
package main

import (
	"fmt"
	"machine"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	adapter = bluetooth.DefaultAdapter
)

func main() {
	// USB CDC serial
	machine.Serial.Configure(machine.UARTConfig{})

	// Give the host time to enumerate the USB serial device.
	time.Sleep(1500 * time.Millisecond)

	fmt.Println("boot: pico2w ble adv loop (fmt logs)")

	fmt.Println("ble: enabling adapter...")
	if err := adapter.Enable(); err != nil {
		fmt.Println("FATAL: adapter.Enable failed:", err)
		for {
			time.Sleep(1 * time.Second)
		}
	}
	fmt.Println("ble: adapter enabled")

	adv := adapter.DefaultAdvertisement()

	// Configure once initially.
	fmt.Println("ble: configuring advertisement...")
	if err := adv.Configure(bluetooth.AdvertisementOptions{
		AdvertisementType: bluetooth.AdvertisingTypeNonConnInd,
		LocalName:         "pico2w-done",
		Interval:          bluetooth.NewDuration(100 * time.Millisecond),
		ManufacturerData: []bluetooth.ManufacturerDataElement{
			{CompanyID: 0xFFFF, Data: []byte{0x01, 0xD0, 0x00, 0x00}},
		},
	}); err != nil {
		fmt.Println("FATAL: adv.Configure failed:", err)
		for {
			time.Sleep(1 * time.Second)
		}
	}
	fmt.Println("ble: advertisement configured")

	var seq byte = 0

	for {
		seq++
		payload := []byte{0x01, 0xD0, seq, 0x00}
		fmt.Printf("loop: seq=%d payload=% X\n", seq, payload)

		// Re-configure each cycle so the payload changes.
		// If this proves flaky on your setup, we can instead keep a fixed payload.
		if err := adv.Configure(bluetooth.AdvertisementOptions{
			AdvertisementType: bluetooth.AdvertisingTypeNonConnInd,
			LocalName:         "pico2w-done",
			Interval:          bluetooth.NewDuration(100 * time.Millisecond),
			ManufacturerData: []bluetooth.ManufacturerDataElement{
				{CompanyID: 0xFFFF, Data: payload},
			},
		}); err != nil {
			fmt.Println("WARN: re-Configure failed (continuing with previous config):", err)
		}

		fmt.Println("ble: adv start")
		if err := adv.Start(); err != nil {
			fmt.Println("ERROR: adv.Start failed:", err)
			fmt.Println("sleep: 2s then retry")
			time.Sleep(2 * time.Second)
			continue
		}

		// Burst advertise: ~6 packets at 100ms interval.
		time.Sleep(600 * time.Millisecond)

		if err := adv.Stop(); err != nil {
			fmt.Println("WARN: adv.Stop failed:", err)
		} else {
			fmt.Println("ble: adv stop")
		}

		fmt.Println("sleep: 2s")
		time.Sleep(2 * time.Second)
	}
}
