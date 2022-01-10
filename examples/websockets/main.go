package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kisielk/monome"
)

type gridmsg struct {
	Cmd  string
	Data []int
}

func main() {
	gmc := make(chan gridmsg, 1)
	handleWs := makeWsHandler(gmc)
	kec := make(chan monome.KeyEvent, 1)
	grid, err := monome.Connect("/gridserver", kec)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(":: grid was connected %v\n", grid.Id())

	localBuffer := monome.NewLEDBuffer(grid.Width(), grid.Height())

	go webGridMessageHandler(gmc, *grid, localBuffer)
	go gridKeyHandler(kec, *grid, localBuffer)

	grid.LEDAll(0)

	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/ws", handleWs)

	go waitForServer()
	panic(http.ListenAndServe(":8080", nil))
}

func makeWsHandler(c chan gridmsg) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		gmc := c
		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		}

		log.Printf(":: web client connected")

		readWs(conn, gmc)
	}
}

func readWs(conn *websocket.Conn, gmc chan gridmsg) {
	for {
		m := gridmsg{}
		err := conn.ReadJSON(&m)
		if err != nil {
			log.Println(err)
			return
		}

		gmc <- m

		if err = conn.WriteJSON(m); err != nil {
			fmt.Println(err)
		}
	}
}

func gridKeyHandler(gke chan monome.KeyEvent, g monome.Grid, localBuffer *monome.LEDBuffer) {
	for {
		m := <-gke
		i := localBuffer.GetIndexFromXY(m.X, m.Y)
	}
}

func webGridMessageHandler(gmc chan gridmsg, g monome.Grid, localBuffer *monome.LEDBuffer) {
	for {
		m := <-gmc
		switch m.Cmd {
		case "levelMap":
			localBuffer.Buf = m.Data
			localBuffer.Render(&g)
		}
	}
}

func waitForServer() {
	for {
		time.Sleep(time.Second * 2)

		log.Println(":: checking webserver status")
		resp, err := http.Get("http://localhost:8080")
		if err != nil {
			log.Println(":: server not ready")
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Println(":: not ok:", resp.StatusCode)
			continue
		}

		break
	}
	fmt.Println(":: server running, opening browser")
	exec.Command("open", "http://localhost:8080").Run()
}
