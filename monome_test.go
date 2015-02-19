package monome

import (
	"fmt"
	"log"
	"testing"
)

func Example() {
	keyEvents := make(chan KeyEvent)
	device, err := Connect("/hello", keyEvents)
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

func TestOscConnection(t *testing.T) {
	_, err := newOscConnection("localhost:12002")
	if err != nil {
		t.Fatal(err)
	}
}
