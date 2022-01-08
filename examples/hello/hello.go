package main

import (
	"fmt"
	"log"
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
	b.LEDAll(0)

	fmt.Println("Hello, press some keys on your grid!")
	fmt.Printf("Connected to monome id: %s, prefix: %s, width: %d, height: %d, rotation: %d\n",
		grid.Id(), grid.Prefix(), grid.Width(), grid.Height(), grid.Rotation())

	go func() {
		<-c
		fmt.Printf("\nShutting Down...\n")
		time.Sleep(1 * time.Second)
		grid.LEDAll(0)
		grid.Close()
		os.Exit(0)
	}()

	for e := range keyEvents {
		fmt.Printf("%+v\n", e)
		s := b.Buf[e.X+(e.Y*grid.Width())]
		if e.State == 0 {
			if s == 15 {
				s = 0
			} else {
				s = 15
			}
			b.LEDLevelSet(e.X, e.Y, s)
			b.Render(grid)
		}
	}
}
