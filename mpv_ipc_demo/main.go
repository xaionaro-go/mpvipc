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

	events, stopListening := conn.NewEventListener()

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
		time.Sleep(3 * time.Second)
		if conn.IsClosed() {
			err := conn.Open()
			if err != nil {
				log.Print(err)
				return
			}
		}

		result, err := conn.Call("set_property", "pause", false)
		if err != nil {
			log.Fatalf("can't set: %s", err)
		} else {
			log.Printf("got result: %v", result)
		}
	}()

	go func() {
		time.Sleep(5 * time.Second)
		stopListening <- struct{}{}
	}()

	for event := range events {
		log.Printf("event: %v", event)
	}
}
