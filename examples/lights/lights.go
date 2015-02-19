package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
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

	width, height := int(device.Width()), int(device.Height())

	ticker := time.NewTicker(2 * time.Second)
	row := make([]byte, width/8)
	for {
		select {
		case e := <-keyEvents:
			if e.State == 1 {
				device.Set(e.X, e.Y, 1)
			}
		case <-ticker.C:
			for i := 0; i < height; i++ {
				for j := 0; j < width; j++ {
					n := byte(rand.Intn(2))
					row[j/8] ^= n << uint(j%8)
				}
				if err := device.Rows(0, i, row...); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		}
	}
}
