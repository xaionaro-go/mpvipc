package main

import (
	"log"

	"github.com/DexterLB/mpv_ipc"
)

func main() {
	conn := mpv_ipc.NewConnection("/tmp/mpv_rpc")
	err := conn.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

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
}
