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
	grid, err := monome.Connect("/lights", keyEvents)
	if err != nil {
		log.Fatal(err)
	}
	defer grid.Close()
	fmt.Printf("Connected to monome id: %s, prefix: %s, width: %d, height: %d, rotation: %d\n",
		grid.Id(), grid.Prefix(), grid.Width(), grid.Height(), grid.Rotation())

	width, height := grid.Width(), grid.Height()
	maxValue := height * 16 // 16 brightness levels per cell

	faders := make([]chan int, width)
	for i := range faders {
		ch := make(chan int)
		go func(ch chan int, x int) {
			value, nextValue := 0, 0
			ticker := time.NewTicker(10 * time.Millisecond)
			for {
				select {
				case <-c:
					fmt.Printf("\nShutting Down...\n")
					time.Sleep(1 * time.Second)
					grid.LEDAll(0)
					grid.Close()
					os.Exit(0)
				case v := <-ch:
					nextValue = v
				case <-ticker.C:
					if value < nextValue {
						value++
					} else if value > nextValue {
						value--
					} else {
						continue
					}
					col := make([]int, height)
					for i := 0; i < height; i++ {
						level := value - i*16
						if level > 15 {
							level = 15
						} else if level < 0 {
							level = 0
						}
						col[i] = level
					}
					grid.LEDLevelCol(x, 0, col)
				}
			}
		}(ch, i)
		faders[i] = ch
		ch <- rand.Intn(maxValue)
	}

	for {
		select {
		case e := <-keyEvents:
			if e.State == 1 {
				faders[e.X] <- (e.Y + 1) * 16
			}
		}
	}
}
