package main

import (
	"fmt"
	"log"

	"github.com/kisielk/monome"
)

func main() {
	keyEvents := make(chan monome.KeyEvent)
	device, err := monome.Connect(keyEvents)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Close()
	for e := range keyEvents {
		fmt.Printf("%+v\n", e)
		device.Set(e.X, e.Y, e.State)
	}
}
