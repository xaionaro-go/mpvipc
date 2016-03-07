// Package mpv_ipc provides an interface for communicating with the mpv media
// player via it's JSON IPC interface
package mpv_ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type Connection struct {
	client     net.Conn
	socketName string

	lastRequest     uint
	waitingRequests map[uint]chan *commandResult

	lastListener   uint
	eventListeners map[uint]chan *Event

	lock *sync.Mutex
}

type Event struct {
	Name   string `json:"event"`
	Reason string `json:"reason"`
	Prefix string `json:"prefix"`
	Level  string `json:"level"`
	Text   string `json:"text"`
}

func NewConnection(socketName string) *Connection {
	return &Connection{
		socketName:      socketName,
		lock:            &sync.Mutex{},
		waitingRequests: make(map[uint]chan *commandResult),
		eventListeners:  make(map[uint]chan *Event),
	}
}

func (c *Connection) Open() error {
	if c.client != nil {
		return fmt.Errorf("already open")
	}
	client, err := net.Dial("unix", c.socketName)
	if err != nil {
		return fmt.Errorf("can't connect to mpv's socket: %s", err)
	}
	c.client = client
	go c.listen()
	return nil
}

func (c *Connection) ListenForEvents(events chan *Event, stop chan struct{}) {
	c.lock.Lock()
	c.lastListener++
	id := c.lastListener
	c.eventListeners[id] = events
	c.lock.Unlock()

	<-stop

	c.lock.Lock()
	delete(c.eventListeners, id)
	c.lock.Unlock()

	close(events)
}

func (c *Connection) NewEventListener() (chan *Event, chan struct{}) {
	events := make(chan *Event)
	stop := make(chan struct{})
	go c.ListenForEvents(events, stop)
	return events, stop
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

func (c *Connection) Close() error {
	if c.client != nil {
		err := c.client.Close()
		if err != nil {
			return err
		}
		c.client = nil
	}
	return nil
}

func (c *Connection) IsClosed() bool {
	return c.client == nil
}

func (c *Connection) sendCommand(id uint, arguments ...interface{}) error {
	if c.client == nil {
		return fmt.Errorf("trying to send command on closed mpv client")
	}
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

type commandRequest struct {
	Arguments []interface{} `json:"command"`
	Id        uint          `json:"request_id"`
}

type commandResult struct {
	Status string      `json:"error"`
	Data   interface{} `json:"data"`
	Id     uint        `json:"request_id"`
}

func (c *Connection) checkResult(data []byte) {
	result := &commandResult{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return // skip malformed data
	} else {
		if result.Status == "" {
			return // not a result
		}
		c.lock.Lock()
		request, ok := c.waitingRequests[result.Id]
		c.lock.Unlock()
		if ok {
			request <- result
		}
	}
}

func (c *Connection) checkEvent(data []byte) {
	event := &Event{}
	err := json.Unmarshal(data, &event)
	if err != nil {
		return // skip malformed data
	} else {
		if event.Name == "" {
			return // not an event
		}
		c.lock.Lock()
		for listenerId := range c.eventListeners {
			listener := c.eventListeners[listenerId]
			go func() {
				listener <- event
			}()
		}
		c.lock.Unlock()
	}
}

func (c *Connection) listen() {
	scanner := bufio.NewScanner(c.client)
	for scanner.Scan() {
		data := scanner.Bytes()
		c.checkEvent(data)
		c.checkResult(data)
	}
	c.Close()
}
