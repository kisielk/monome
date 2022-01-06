package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/kisielk/monome"
)

func main() {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	keyEvents := make(chan monome.KeyEvent)
	grid, err := monome.Connect("/hello", keyEvents)
	if err != nil {
		log.Fatal(err)
	}

	b := monome.NewLEDBuffer(grid.Width(), grid.Height())
	rand.Seed(time.Now().UnixNano())

	// fill buffer with random values from 0-15
	for i := range b.Buf {
		b.Buf[i] = rand.Intn(15-0) + 0
	}

	b.Render(grid)

	fmt.Printf("Connected to monome id: %s, prefix: %s, width: %d, height: %d, rotation: %d\n",
		grid.Id(), grid.Prefix(), grid.Width(), grid.Height(), grid.Rotation())

	go func() {
		<-c
		fmt.Printf("\nCleaning up\n\n")
		time.Sleep(1 * time.Second)
		grid.LEDAll(0)
		grid.Close()
		os.Exit(1)
	}()

	for e := range keyEvents {
		fmt.Printf("%+v\n", e)
		if e.State == 1 {
			grid.LEDSet(e.X, e.Y, e.State)
		} else {
			grid.LEDLevelSet(e.X, e.Y, b.Buf[(e.Y*grid.Width())+e.X])
		}
	}

}
