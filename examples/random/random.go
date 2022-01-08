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

func makeRandomLights(b *monome.LEDBuffer, d *monome.Grid) {
	rand.Seed(time.Now().UnixNano())

	// fill buffer with random values from 0-15
	for i := range b.Buf {
		b.Buf[i] = rand.Intn(15)
	}
	b.Render(d)
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	keyEvents := make(chan monome.KeyEvent)
	grid, err := monome.Connect("/randomlights", keyEvents)
	if err != nil {
		log.Fatal(err)
	}

	b := monome.NewLEDBuffer(grid.Width(), grid.Height())

	go func() {
		<-c
		fmt.Printf("\nShutting Down...\n")
		time.Sleep(1 * time.Second)
		grid.LEDAll(0)
		grid.Close()
		os.Exit(0)
	}()

	for {
		time.Sleep(500 * time.Millisecond)
		makeRandomLights(b, grid)
	}
}
