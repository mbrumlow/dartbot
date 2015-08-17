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

	"golang.org/x/net/websocket"
)

const (
	Signal = 1 << iota
	TrackPower
	Video
	StartVideo
	EndVideo
)

type JsonEvent struct {
	Type  int
	Event string
}

type Power struct {
	Left  uint8
	Right uint8
}

type videoClient struct {
	WS   *websocket.Conn
	done chan bool
}

var clientMU sync.Mutex
var clients = make(map[*videoClient]bool)

var robothostport = flag.String("host", "", "host port of dartbot")

func main() {

	flag.Parse()

	events := make(chan JsonEvent, 10)

	if *robothostport == "" {
		flag.PrintDefaults()
		log.Fatal("Plase provide a host port.\n")
	}

	url := fmt.Sprintf("ws://%v/control", *robothostport)
	ref := fmt.Sprintf("http://%v/", *robothostport)

	ws, err := websocket.Dial(url, "", ref)
	if err != nil {
		log.Fatal("Failed to connect to dartbot: %v\n", err.Error())
	}

	// TODO - reconnect in loop.

	go startHttp(events)
	go handleRobotEvents(ws)

	for {
		event := <-events
		websocket.JSON.Send(ws, &event)
	}
}

var mu sync.Mutex
var v *websocket.Conn = nil

func handleRobotEvents(ws *websocket.Conn) {

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

func sendVideoToClients(data []byte) {

	var c = make(map[*videoClient]bool)

	clientMU.Lock()
	for k, v := range clients {
		c[k] = v
	}
	clientMU.Unlock()

	for v := range c {
		if err := websocket.Message.Send(v.WS, data); err != nil {
			log.Printf("ERROR: Failed to send video data client: %v.\n", err.Error())
			removeClient(v)
		}
	}
}

func startHttp(events chan JsonEvent) {

	http.HandleFunc("/power", func(w http.ResponseWriter, r *http.Request) {
		powerHandler(w, r, events)
	})

	http.Handle("/video", websocket.Handler(func(ws *websocket.Conn) {
		videoHandler(ws, events)
	}))

	fs := http.FileServer(http.Dir("webroot"))
	http.Handle("/", http.StripPrefix("/", fs))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func videoHandler(ws *websocket.Conn, events chan JsonEvent) {

	if !sendHeader(ws) {
		return
	}

	startEvent := JsonEvent{Type: StartVideo}
	events <- startEvent

	v := addClient(ws)
	<-v.done

	clientMU.Lock()
	defer clientMU.Unlock()

	if len(clients) >= 0 {
		return
	}

	endEvent := JsonEvent{Type: EndVideo}
	events <- endEvent

}

func removeClient(v *videoClient) {
	clientMU.Lock()
	defer clientMU.Unlock()
	delete(clients, v)
	v.done <- true
}

func addClient(ws *websocket.Conn) *videoClient {
	clientMU.Lock()
	defer clientMU.Unlock()

	v := &videoClient{WS: ws}
	v.done = make(chan bool)
	clients[v] = true

	return v
}

func sendHeader(ws *websocket.Conn) bool {

	mu.Lock()
	defer mu.Unlock()

	bb := new(bytes.Buffer)
	bb.Write([]byte("jsmp"))
	binary.Write(bb, binary.BigEndian, uint16(1280))
	binary.Write(bb, binary.BigEndian, uint16(720))

	if err := websocket.Message.Send(ws, bb.Bytes()); err != nil {
		log.Printf("ERROR: Failed to send video header to client: %v.\n", err.Error())
		return false
	}

	return true
}

func powerHandler(w http.ResponseWriter, r *http.Request, events chan JsonEvent) {

	log.Println("Powerhandler.")

	jsonBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read body: %v", err.Error())
		http.Error(w, "Failed to read body.", 500)
		return
	}

	var power Power
	if err := json.Unmarshal(jsonBytes, &power); err != nil {
		log.Printf("ERROR: Failed to unmarshal power: %v\n", err.Error())
		http.Error(w, "Failed to unmarshal power", 400)
		return
	}

	event := JsonEvent{Type: TrackPower, Event: string(jsonBytes)}

	events <- event

}
