package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

const (
	maxVideo  = 200
	maxEvents = 250
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

type JsonEvent struct {
	Type  int
	Event string
}

type Action struct {
	Time   string
	Name   string
	Action string
}

type Power struct {
	Left  uint8
	Right uint8
}

type Chat struct {
	Name string
	Text string
}

var (
	clientMu     sync.RWMutex
	eventClients = make(map[chan JsonEvent]*websocket.Conn)
	videoClients = make(map[chan []byte]*websocket.Conn)
)

var robothostport = flag.String("host", "", "host port of dartbot")

func main() {

	flag.Parse()

	if *robothostport == "" {
		flag.PrintDefaults()
		log.Fatal("Plase provide a host port.\n")
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

			// clean out peding events
			for i := len(events); i > 0; i-- {
				<-events
			}

			time.Sleep(1 * time.Second)
			continue
		}

		// clean out peding events
		for i := len(events); i > 0; i-- {
			<-events
		}

		go handleRobotEvents(ws)

		for {
			event := <-events
			if err := websocket.JSON.Send(ws, &event); err != nil {
				log.Printf("ERROR: Failed to send event to robot: %v.\n", err.Error())
				break
			}
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

	http.HandleFunc("/power", func(w http.ResponseWriter, r *http.Request) {
		powerHandler(w, r, events)
	})

	http.HandleFunc("/chat", chatHandler)

	http.Handle("/video", websocket.Handler(clientVideoHandler))
	http.Handle("/events", websocket.Handler(clientEventHandler))

	fs := http.FileServer(http.Dir("webroot2"))
	http.Handle("/", http.StripPrefix("/", fs))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func sendClientEvent(je JsonEvent) {

	clientMu.RLock()
	defer clientMu.RUnlock()

	for e, ws := range eventClients {

		if len(e) > maxEvents-(maxEvents/10) {
			log.Printf("INFO Dropping video frames on client: %v\n", ws.Request().RemoteAddr)
			for len(e) != 0 {
				<-e
			}
		}

		e <- je
	}
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
		powerEvent([]byte(ev.Event))
	default:
		log.Printf("ERROR: Not sending unknown event type (%v) to server.\n", ev.Type)
	}

}

func powerEvent(jsonBytes []byte) {

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

	je := JsonEvent{Type: ActionEvent, Event: string(jsonBytes)}

	sendClientEvent(je)

}

func robotDownEvent() {

	a := Action{Time: formatedTime(), Name: "SYSTEM", Action: "OFFLINE"}

	jsonBytes, err := json.Marshal(a)
	if err != nil {
		log.Printf("ERROR: Failed to marshal json: %v.\n", err.Error())
	}

	je := JsonEvent{Type: ActionEvent, Event: string(jsonBytes)}

	sendClientEvent(je)
}

func clientEventHandler(ws *websocket.Conn) {

	eventChan := make(chan JsonEvent, maxEvents)
	addEventClient(eventChan, ws)
	defer removeEventClient(eventChan)

	wsLogInfo(ws, "Event client connected.")
	defer wsLogInfo(ws, "Event client disconnected.")

	for {
		event := <-eventChan
		if err := websocket.JSON.Send(ws, &event); err != nil {
			wsLogError(ws, err.Error())
			return
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

func addEventClient(e chan JsonEvent, ws *websocket.Conn) {
	clientMu.Lock()
	defer clientMu.Unlock()
	eventClients[e] = ws
}

func removeEventClient(e chan JsonEvent) {
	clientMu.Lock()
	defer clientMu.Unlock()
	delete(eventClients, e)
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

func powerHandler(w http.ResponseWriter, r *http.Request, events chan JsonEvent) {

	logInfo(r, "Power handler")

	jsonBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logError(r, fmt.Sprintf("Failed to read body: %v", err.Error()))
		http.Error(w, "Failed to read body.", 500)
		return
	}

	var power Power
	if err := json.Unmarshal(jsonBytes, &power); err != nil {
		logError(r, fmt.Sprintf("Failed to unmarshal power: %v", err.Error()))
		http.Error(w, "Failed to unmarshal power.", 400)
		return
	}

	event := JsonEvent{Type: TrackPower, Event: string(jsonBytes)}
	events <- event
}

func chatHandler(w http.ResponseWriter, r *http.Request) {

	logInfo(r, "Chat handler")

	jsonBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logError(r, fmt.Sprintf("Failed to read body: %v", err.Error()))
		http.Error(w, "Failed to read body.", 500)
		return
	}

	var chat Chat
	if err := json.Unmarshal(jsonBytes, &chat); err != nil {
		logError(r, fmt.Sprintf("Failed to unmarshal chat: %v", err.Error()))
		http.Error(w, "Failed to unmarshal chat.", 400)
		return
	}

	a := Action{Time: formatedTime(), Name: chat.Name, Action: chat.Text}
	jsonBytes, err = json.Marshal(a)
	if err != nil {
		log.Printf("ERROR: Failed to marshal json: %v.\n", err.Error())
	}

	je := JsonEvent{Type: ChatEvent, Event: string(jsonBytes)}

	sendClientEvent(je)
}

func logInfo(r *http.Request, msg string) {
	log.Printf("INFO - %v - %v\n", r.RemoteAddr, msg)
}

func logError(r *http.Request, msg string) {
	log.Printf("ERROR - %v - %v\n", r.RemoteAddr, msg)
}

func wsLogInfo(ws *websocket.Conn, msg string) {
	wsLog(ws, fmt.Sprintf("INFO - %v - %v\n", ws.Request().RemoteAddr, msg))
}

func wsLogError(ws *websocket.Conn, msg string) {
	wsLog(ws, fmt.Sprintf("ERROR - %v - %v\n", ws.Request().RemoteAddr, msg))
}

func wsLog(ws *websocket.Conn, msg string) {
	log.Printf("%v", msg)
}

func formatedTime() string {
	return time.Now().Format("03:04:05.000")
}
