package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/kisielk/monome"
)

func main() {
	log.SetFlags(0)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage :: websockets host:port, i.e. $ websockets localhost:55555")
	}

	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		return err
	}
	log.Printf("success :: listening on http://%v", l.Addr())

	keyEvents := make(chan monome.KeyEvent)
	grid, err := monome.Connect("/gridserver", keyEvents)
	if err != nil {
		log.Fatal(err)
	}

	gs := newGridServer(keyEvents, grid)
	s := &http.Server{
		Handler:      gs,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	errc := make(chan error, 1)
	go func() {
		errc <- s.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case <-sigs:
		fmt.Printf("\nclosing :: shutting down gridServer..\n")
		time.Sleep(1 * time.Second)
		grid.LEDAll(0)
		grid.Close()
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}
