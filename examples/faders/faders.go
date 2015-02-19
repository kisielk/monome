package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/kisielk/monome"
)

func main() {
	keyEvents := make(chan monome.KeyEvent)
	device, err := monome.Connect("/lights", keyEvents)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Close()

	// Wait for monome to send its info.
	time.Sleep(1 * time.Second)
	fmt.Printf("Connected to monome id: %s, prefix: %s, width: %d, height: %d, rotation: %d\n",
		device.Id(), device.Prefix(), device.Width(), device.Height(), device.Rotation())

	width, height := device.Width(), device.Height()
	maxValue := height * 16 // 16 brightness levels per cell

	faders := make([]chan int, width)
	for i := range faders {
		ch := make(chan int)
		go func(ch chan int, x int) {
			value, nextValue := 0, 0
			ticker := time.NewTicker(10 * time.Millisecond)
			for {
				select {
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
					device.LevelCols(x, 0, col)
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
