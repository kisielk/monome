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
	grid, err := monome.Connect("/lights", keyEvents)
	if err != nil {
		log.Fatal(err)
	}
	defer grid.Close()
	fmt.Printf("Connected to monome id: %s, prefix: %s, width: %d, height: %d, rotation: %d\n",
		grid.Id(), grid.Prefix(), grid.Width(), grid.Height(), grid.Rotation())

	width, height := grid.Width(), grid.Height()

	ticker := time.NewTicker(2 * time.Second)
	row := make([]byte, width/8)
	for {
		select {
		case e := <-keyEvents:
			if e.State == 1 {
				grid.LEDSet(e.X, e.Y, 1)
			}
		case <-ticker.C:
			for i := 0; i < height; i++ {
				for j := 0; j < width; j++ {
					n := byte(rand.Intn(2))
					row[j/8] ^= n << uint(j%8)
				}
				if err := grid.LEDRow(0, i, row...); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		}
	}
}
