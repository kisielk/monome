package main

import (
	"fmt"
	"log"

	"github.com/kisielk/monome"
)

func main() {
	keyEvents := make(chan monome.KeyEvent)
	grid, err := monome.Connect("/hello", keyEvents)
	if err != nil {
		log.Fatal(err)
	}
	defer grid.Close()

	fmt.Printf("Connected to monome id: %s, prefix: %s, width: %d, height: %d, rotation: %d\n",
		grid.Id(), grid.Prefix(), grid.Width(), grid.Height(), grid.Rotation())
	for e := range keyEvents {
		fmt.Printf("%+v\n", e)
		grid.LEDSet(e.X, e.Y, e.State)
	}
}
