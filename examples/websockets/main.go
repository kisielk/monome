package main

import (
	"fmt"
	"log"
	"net/http"

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
	keyEvents := make(chan monome.KeyEvent, 1)
	grid, err := monome.Connect("/gridserver", keyEvents)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(":: grid was connected %v\n", grid.Id())
	go gridMessageHandler(gmc, *grid)

	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/ws", handleWs)

	panic(http.ListenAndServe(":8080", nil))
}

func makeWsHandler(c chan gridmsg) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		gmc := c
		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		}

		log.Printf(":: client connected")

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

func gridMessageHandler(gmc chan gridmsg, g monome.Grid) {
	for {
		m := <-gmc
		fmt.Printf("%s, %d, %d\n", m.Cmd, m.Data[0], m.Data[1])
		g.LEDSet(m.Data[0], m.Data[1], 1)
	}
}

// sigs := make(chan os.Signal, 1)
// signal.Notify(sigs, os.Interrupt)

// select {
// case err := <-errc:
// 	log.Printf("failed to serve: %v", err)
// case <-sigs:
// 	fmt.Printf("\nclosing :: shutting down gridServer..\n")
// 	time.Sleep(1 * time.Second)
// 	grid.LEDAll(0)
// 	grid.Close()
// 	os.Exit(0)
// }

// ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
// defer cancel()

// return s.Shutdown(ctx)
// }
