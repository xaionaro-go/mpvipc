package main

import (
	"log"
	"time"

	"github.com/DexterLB/mpv_ipc"
)

func main() {
	conn := mpv_ipc.NewConnection("/tmp/mpv_rpc")
	err := conn.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	stopListening := make(chan struct{})
	events := make(chan *mpv_ipc.Event)

	go conn.ListenForEvents(stopListening, events)

	result, err := conn.Call("set_property", "pause", true)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("got result: %v", result)
	}
	result, err = conn.Call("get_property", "pause")
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("got result: %v", result)
	}

	go func() {
		time.Sleep(5 * time.Second)
		stopListening <- struct{}{}
	}()

	for event := range events {
		log.Printf("event: %v", event)
	}
}
