package main

import (
	"log"
	"time"

	"github.com/DexterLB/mpvipc"
)

func main() {
	conn := mpvipc.NewConnection("/tmp/mpv_rpc")
	err := conn.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	events, stopListening := conn.NewEventListener()

	path, err := conn.Get("path")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("path: %s", path)

	go func() {
		time.Sleep(5 * time.Second)
		stopListening <- struct{}{}
	}()

	for event := range events {
		log.Printf("event: %v", event)
	}
}
