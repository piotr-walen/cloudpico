// BLE beacon for Pico 2 W. Advertises continuously so the gateway can discover it.
//
// Build and flash (use pico2-w for the wireless board):
//   tinygo flash -target=pico2-w .
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

	fmt.Println("boot: pico2w BLE beacon (continuous adv)")

	fmt.Println("ble: enabling adapter...")
	if err := adapter.Enable(); err != nil {
		fmt.Println("FATAL: adapter.Enable failed:", err)
		for {
			time.Sleep(1 * time.Second)
		}
	}
	fmt.Println("ble: adapter enabled")

	adv := adapter.DefaultAdvertisement()

	// Configure once; fixed payload so we don't re-Configure (can be flaky on some stacks).
	payload := []byte{0x01, 0xD0, 0x00, 0x00}
	fmt.Println("ble: configuring advertisement...")
	if err := adv.Configure(bluetooth.AdvertisementOptions{
		AdvertisementType: bluetooth.AdvertisingTypeNonConnInd,
		LocalName:         "pico2w-done",
		Interval:          bluetooth.NewDuration(100 * time.Millisecond),
		ManufacturerData: []bluetooth.ManufacturerDataElement{
			{CompanyID: 0xFFFF, Data: payload},
		},
	}); err != nil {
		fmt.Println("FATAL: adv.Configure failed:", err)
		for {
			time.Sleep(1 * time.Second)
		}
	}
	fmt.Println("ble: advertisement configured")

	fmt.Println("ble: adv start (continuous)")
	if err := adv.Start(); err != nil {
		fmt.Println("FATAL: adv.Start failed:", err)
		for {
			time.Sleep(1 * time.Second)
		}
	}
	fmt.Println("ble: advertising; gateway should see 'pico2w-done'")

	// Keep running and print every 5s so serial shows we're alive.
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		<-tick.C
		fmt.Println("ble: still advertising...")
	}
}
