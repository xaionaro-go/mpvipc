// Package mpv_ipc provides an interface for communicating with the mpv media
// player via it's JSON IPC interface
package mpv_ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
)

type Connection struct {
	client          net.Conn
	socketName      string
	lastRequest     uint
	waitingRequests map[uint]chan *commandResult
	lock            *sync.Mutex
}

func NewConnection(socketName string) *Connection {
	return &Connection{
		socketName:      socketName,
		lock:            &sync.Mutex{},
		waitingRequests: make(map[uint]chan *commandResult),
	}
}

func (c *Connection) Open() error {
	client, err := net.Dial("unix", c.socketName)
	if err != nil {
		return fmt.Errorf("can't connect to mpv's socket: %s", err)
	}
	c.client = client
	go c.listen()
	return nil
}

func (c *Connection) Call(arguments ...interface{}) (interface{}, error) {
	c.lock.Lock()
	c.lastRequest++
	id := c.lastRequest
	resultChannel := make(chan *commandResult)
	c.waitingRequests[id] = resultChannel
	c.lock.Unlock()

	defer func() {
		c.lock.Lock()
		close(c.waitingRequests[id])
		delete(c.waitingRequests, id)
		c.lock.Unlock()
	}()

	err := c.sendCommand(id, arguments...)
	if err != nil {
		return nil, err
	}

	result := <-resultChannel
	if result.Status == "success" {
		return result.Data, nil
	}
	return nil, fmt.Errorf("mpv error: %s", result.Status)
}

func (c *Connection) sendCommand(id uint, arguments ...interface{}) error {
	message := &commandRequest{
		Arguments: arguments,
		Id:        id,
	}
	data, err := json.Marshal(&message)
	if err != nil {
		return fmt.Errorf("can't encode command: %s", err)
	}
	_, err = c.client.Write(data)
	if err != nil {
		return fmt.Errorf("can't write command: %s", err)
	}
	_, err = c.client.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("can't terminate command: %s", err)
	}
	return err
}

func (c *Connection) Close() error {
	err := c.client.Close()
	if err != nil {
		return err
	}
	c.client = nil
	return nil
}

type commandRequest struct {
	Arguments []interface{} `json:"command"`
	Id        uint          `json:"request_id"`
}

type commandResult struct {
	Status string      `json:"error"`
	Data   interface{} `json:"data"`
	Id     uint        `json:"request_id"`
}

func (c *Connection) listen() {
	scanner := bufio.NewScanner(c.client)
	for scanner.Scan() {
		data := scanner.Bytes()
		log.Printf("got some data: %s", string(data))
		result := &commandResult{}
		err := json.Unmarshal(data, &result)
		if err == nil {
			c.lock.Lock()
			request, ok := c.waitingRequests[result.Id]
			c.lock.Unlock()
			if ok {
				request <- result
			}
		} else {
			log.Printf(":( %s", err)
		}
	}
}
