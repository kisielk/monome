package main

import (
	"fmt"
	"log"

	"github.com/kisielk/monome"
)

func main() {
	keyEvents := make(chan monome.KeyEvent)
	device, err := monome.Connect("/hello", keyEvents)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Close()

	fmt.Printf("Connected to monome id: %s, prefix: %s, width: %d, height: %d, rotation: %d\n",
		device.Id(), device.Prefix(), device.Width(), device.Height(), device.Rotation())
	for e := range keyEvents {
		fmt.Printf("%+v\n", e)
		device.LEDSet(e.X, e.Y, e.State)
	}
}
