package main

import (
	"machine"
	"time"
)

func put(s string) {
	machine.Serial.Write([]byte(s))
}

func main() {
	time.Sleep(2 * time.Second)

	put("hello from pico2-w (tinygo)\r\n")
	for {
		put("tick\r\n")
		time.Sleep(1 * time.Second)
	}
}
