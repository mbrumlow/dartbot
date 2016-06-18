package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

const (
	maxVideo  = 100
	maxEvents = 10
)

const (
	Signal = 1 << iota
	TrackPower
	Video
	StartVideo
	EndVideo
	ActionEvent
	ChatEvent
)

const (
	_                = iota
	AuthOK           = iota
	AuthUserInUse    = iota
	AuthPassRequired = iota
	AuthBadPass      = iota
	AuthBadName      = iota
)

type JsonEvent struct {
	Name  string
	Type  int
	Event string
}

type Action struct {
	Time   string
	Action string
}

type Power struct {
	Left  int16
	Right int16
}

type Chat struct {
	Auth string
	Text string
}

type AuthEvent struct {
	Name string
	Auth string
}

type Client struct {
	To     chan JsonEvent
	From   chan JsonEvent
	Name   string
	Active bool
	ws     *websocket.Conn
}

var (
	clientMu     sync.RWMutex
	clients      = make(map[string]map[*Client]interface{})
	eventClients = make(map[chan JsonEvent]*websocket.Conn)
	videoClients = make(map[chan []byte]*websocket.Conn)
)

var robothostport = flag.String("host", "", "host port of dartbot")

func main() {

	flag.Parse()

	if *robothostport == "" {
		flag.PrintDefaults()
		log.Fatal("Plase provide a host:port.\n")
	}

	url := fmt.Sprintf("ws://%v/control", *robothostport)
	ref := fmt.Sprintf("http://%v/", *robothostport)

	events := make(chan JsonEvent, 1000)

	go startHttp(events)

	for {

		ws, err := websocket.Dial(url, "", ref)
		if err != nil {
			log.Printf("ERROR: Failed to connect to dartbot: %v\n", err.Error())
			robotDownEvent()

			// Clean out pending events.
			for i := len(events); i > 0; i-- {
				<-events
			}

			time.Sleep(1 * time.Second)
			continue
		}

		// Clean out pending events.
		for i := len(events); i > 0; i-- {
			<-events
		}

		go handleRobotEvents(ws)

		for {
			event := <-events
			name := event.Name
			event.Name = ""
			if err := websocket.JSON.Send(ws, &event); err != nil {
				log.Printf("ERROR: Failed to send event to robot: %v.\n", err.Error())
				break
			}
			event.Name = name
			sendEventToClient(event)
		}

	}
}

func handleRobotEvents(ws *websocket.Conn) {

	defer ws.Close()

	for {
		var ev JsonEvent
		if err := websocket.JSON.Receive(ws, &ev); err != nil {
			log.Printf("ERROR: failed to recive event from robot: %v\n", err.Error())
			return
		}

		switch ev.Type {
		case Video:
			decodeVideo(ev.Event)
		default:
			log.Println("ERROR: Recived unknown event (%v) from robot.\n", ev.Type)
		}
	}
}

func decodeVideo(s string) {

	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		log.Printf("ERROR: Failed to decode video buffer: %v\n", err.Error())
		return
	}

	sendVideoToClients(decoded)
}

func startHttp(events chan JsonEvent) {

	http.Handle("/video", websocket.Handler(clientVideoHandler))
	http.Handle("/client", websocket.Handler(func(ws *websocket.Conn) {
		clientHandler(ws, events)
	}))

	fs := http.FileServer(http.Dir("webroot"))
	http.Handle("/", http.StripPrefix("/", fs))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func sendVideoToClients(d []byte) {

	clientMu.RLock()
	defer clientMu.RUnlock()

	for v, ws := range videoClients {

		if len(v) > maxVideo-(maxVideo/10) {
			log.Printf("INFO Dropping video frames on client: %v\n", ws.Request().RemoteAddr)
			for len(v) != 0 {
				<-v
			}
		}

		v <- d
	}
}

func sendEventToClient(ev JsonEvent) {

	switch ev.Type {
	case TrackPower:
		powerEvent(ev.Name, []byte(ev.Event))
	default:
		log.Printf("ERROR: Not sending unknown event type (%v) to server.\n", ev.Type)
	}

}

func powerEvent(name string, jsonBytes []byte) {

	var p Power
	if err := json.Unmarshal(jsonBytes, &p); err != nil {
		log.Printf("ERROR: Failed to unmarshal power: %v\n", err.Error())
		return
	}

	a := Action{Time: formatedTime(), Action: fmt.Sprintf("-- POWER(%v,%v) --", p.Left, p.Right)}

	jsonBytes, err := json.Marshal(a)
	if err != nil {
		log.Printf("ERROR: Failed to marshal json: %v.\n", err.Error())
	}

	je := JsonEvent{Name: name, Type: ActionEvent, Event: string(jsonBytes)}

	sendToAll(je)

}

func robotDownEvent() {

	a := Action{Time: formatedTime(), Action: "OFFLINE"}

	jsonBytes, err := json.Marshal(a)
	if err != nil {
		log.Printf("ERROR: Failed to marshal json: %v.\n", err.Error())
	}

	je := JsonEvent{Name: "SYSTEM", Type: ActionEvent, Event: string(jsonBytes)}

	sendToAll(je)
}

func jsonEvent(t int, v interface{}, name string) (JsonEvent, error) {

	jb, err := json.Marshal(v)
	if err != nil {
		return JsonEvent{}, err
	}

	je := JsonEvent{Name: name, Type: t, Event: string(jb)}
	return je, nil
}

func clientEventReader(c *Client) {
	for {
		var je JsonEvent
		if err := websocket.JSON.Receive(c.ws, &je); err != nil {
			wsLogErrorf(c.ws, "Error reading event: %v", err)
			close(c.From)
			return
		}
		c.From <- je
	}
}

func addClient(c *Client, authenticated bool) bool {

	clientMu.Lock()
	defer clientMu.Unlock()

	m, ok := clients[c.Name]
	if ok && !authenticated {
		return false
	}

	if !ok {
		m = make(map[*Client]interface{})
		clients[c.Name] = m
	}

	m[c] = nil
	log.Printf("Adding client %v:%p\n", c.Name, c)

	return true
}

func delClient(c *Client) {

	log.Printf("Deleting client %v:%p\n", c.Name, c)

	clientMu.Lock()
	defer clientMu.Unlock()

	m, ok := clients[c.Name]
	if !ok {
		return
	}

	delete(m, c)

	if len(m) == 0 {
		delete(clients, c.Name)
	}

}

func clientHandler(ws *websocket.Conn, events chan JsonEvent) {

	defer ws.Close()
	wsLogInfo(ws, "Connected.")
	defer wsLogInfo(ws, "Disconnected.")

	client := &Client{
		To:   make(chan JsonEvent, maxEvents),
		From: make(chan JsonEvent, 1),
		ws:   ws,
	}

	for {

		var authEvent AuthEvent
		err := websocket.JSON.Receive(ws, &authEvent)
		if err != nil {
			wsLogInfof(ws, "Error receiving auth event: %v", err.Error())
			return
		}

		wsLogInfof(ws, "Recived auth for '%v'", authEvent.Name)

		if authEvent.Name == "" {
			je, err := jsonEvent(AuthBadName, "Invalid username.", "")
			if err != nil {
				wsLogErrorf(ws, "Failed to create AuthUserInUse event: %v", err)
				return
			}
			if err := websocket.JSON.Send(ws, &je); err != nil {
				wsLogErrorf(ws, "Failed to send AuthBadName event: %v", err)
				return
			}
			continue
		}

		authenticated := false
		if len(authEvent.Auth) > 0 {
			// TODO authenticate

		}

		if !authenticated {
			// TODO: Check if name is registered.
		}

		client.Name = authEvent.Name
		if addClient(client, authenticated) != true {

			je, err := jsonEvent(AuthUserInUse, "Username already in use.", "")
			if err != nil {
				wsLogErrorf(ws, "Failed to create AuthUserInUse event: %v", err)
				return
			}

			if err := websocket.JSON.Send(ws, &je); err != nil {
				wsLogErrorf(ws, "Failed to send AuthUserInUse event: %v", err)
				return
			}
		} else {
			defer delClient(client)
			break
		}

	}

	client.logInfof("Authenticated.")

	if err := func() error {
		je, err := jsonEvent(AuthOK, "Authenticated.", fixName(client.Name))
		if err != nil {
			return fmt.Errorf("Failed to create AuthUserInUse event: %v", err)
		}

		if err := websocket.JSON.Send(ws, &je); err != nil {
			return fmt.Errorf("Failed to send AuthUserInUse event: %v", err)

		}
		return nil
	}(); err != nil {
		wsLogErrorf(ws, err.Error())
		return
	}

	// TODO - make client active / sync
	// This should populate the clients chat back log and set current robot state.

	go clientEventReader(client)

	for {
		select {
		case event := <-client.To:
			if err := websocket.JSON.Send(ws, &event); err != nil {
				wsLogErrorf(ws, "Error sending event: %v", err)
				return
			}
		case clientEvent, ok := <-client.From:
			if !ok {
				return
			}
			client.handleEvent(clientEvent, events)
		}
	}

}

func (c *Client) handleEvent(je JsonEvent, events chan JsonEvent) {

	switch je.Type {
	case ChatEvent:
		c.handleChatEvent(je)
	case TrackPower:
		c.handleTrackPowerEvent(je, events)
	default:
		log.Printf("Recived unknown event (%v)\n", je.Type)
	}

}

func (c *Client) handleChatEvent(e JsonEvent) {

	c.logPrefixf("CHAT", "%v\n", e.Event)

	a := Action{Time: formatedTime(), Action: e.Event}
	je, err := jsonEvent(ChatEvent, a, fixName(c.Name))
	if err != nil {
		log.Printf("Failed to create jsonEvent: %v", err)
		return
	}

	sendToAll(je)
}

func (c *Client) handleTrackPowerEvent(e JsonEvent, events chan JsonEvent) {

	// Sanity check, decode and encode before sending it to the robot.
	var p Power
	if err := json.Unmarshal([]byte(e.Event), &p); err != nil {
		log.Printf("Failed decode TrackPower: %v\n", err)
		return
	}

	c.logPrefixf("POWER", "%v,%v\n", p.Left, p.Right)

	je, err := jsonEvent(TrackPower, p, fixName(c.Name))
	if err != nil {
		log.Printf("Failed to create jsonEvent: %v", err)
		return
	}

	events <- je
}

func (c *Client) logPrefixf(prefix, format string, a ...interface{}) {

	remoteAddr := "0"
	if c.ws != nil {
		remoteAddr = c.ws.Request().RemoteAddr
	}

	msg := fmt.Sprintf(format, a...)
	log.Printf("%v:%v[%p] - %v - %v", remoteAddr, c.Name, c, prefix, msg)

}

func (c *Client) logInfof(format string, a ...interface{}) {
	c.logPrefixf("INFO", format, a...)
}

func sendToAll(je JsonEvent) {

	clientMu.RLock()
	defer clientMu.RUnlock()

	for _, m := range clients {
		for c, _ := range m {
			if len(c.To) > maxEvents-(maxEvents/10) {
				wsLogInfof(c.ws, "Dropping event.")
				for len(c.To) != 0 {
					<-c.To
				}
			}
			c.To <- je
		}
	}
}

func clientVideoHandler(ws *websocket.Conn) {

	if err := sendJSMPHeader(ws); err != nil {
		log.Printf("INFO: Video client ended: %v.\n", err.Error())
		return
	}

	videoChan := make(chan []byte, maxVideo)
	addVideoClient(videoChan, ws)
	defer removeVideoClient(videoChan)

	wsLogInfo(ws, "Video client connected.")
	defer wsLogInfo(ws, "Video client disconnected.")

	for {
		data := <-videoChan
		if err := websocket.Message.Send(ws, data); err != nil {
			wsLogError(ws, err.Error())
			return
		}
	}
}

func addVideoClient(v chan []byte, ws *websocket.Conn) {
	clientMu.Lock()
	defer clientMu.Unlock()
	videoClients[v] = ws
}

func removeVideoClient(v chan []byte) {
	clientMu.Lock()
	defer clientMu.Unlock()
	delete(videoClients, v)
}

func sendJSMPHeader(ws *websocket.Conn) error {

	bb := new(bytes.Buffer)
	bb.Write([]byte("jsmp"))
	binary.Write(bb, binary.BigEndian, uint16(640))
	binary.Write(bb, binary.BigEndian, uint16(480))

	if err := websocket.Message.Send(ws, bb.Bytes()); err != nil {
		return err
	}

	return nil
}

func fixName(name string) string {
	if len(name) > 8 {
		return name[0:8]
	}
	name = name + strings.Repeat(" ", 8-len(name))
	return name
}

// TODO -- fix this logging stuff, its nasty.

func logInfo(r *http.Request, msg string) {
	log.Printf("INFO - %v - %v\n", r.RemoteAddr, msg)
}

func logError(r *http.Request, msg string) {
	log.Printf("ERROR - %v - %v\n", r.RemoteAddr, msg)
}

func wsLogInfo(ws *websocket.Conn, msg string) {
	wsLog(ws, fmt.Sprintf("INFO - %v - %v\n", ws.Request().RemoteAddr, msg))
}

func wsLogInfof(ws *websocket.Conn, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	wsLog(ws, fmt.Sprintf("INFO - %v - %v\n", ws.Request().RemoteAddr, msg))
}

func wsLogError(ws *websocket.Conn, msg string) {
	wsLog(ws, fmt.Sprintf("ERROR - %v - %v\n", ws.Request().RemoteAddr, msg))
}

func wsLogErrorf(ws *websocket.Conn, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	wsLog(ws, fmt.Sprintf("ERROR - %v - %v\n", ws.Request().RemoteAddr, msg))
}

func wsLog(ws *websocket.Conn, msg string) {
	log.Printf("%v", msg)
}

func formatedTime() string {
	return time.Now().Format("03:04:05.000")
}
