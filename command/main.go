package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

type JsonEvent struct {
	Type  int
	Event string
}

type Power struct {
	Left  uint8
	Right uint8
}

func main() {

	events := make(chan JsonEvent, 10)

	ws, err := websocket.Dial("ws://10.0.0.21:8080/control", "", "http://10.0.0.21/")
	if err != nil {
		log.Fatal("Failed to connect to dartbot: %v\n", err.Error())
	}

	go startHttp(events)

	for {
		event := <-events
		websocket.JSON.Send(ws, &event)
	}
}

func startHttp(events chan JsonEvent) {

	http.HandleFunc("/power", func(w http.ResponseWriter, r *http.Request) {
		powerHandler(w, r, events)
	})

	fs := http.FileServer(http.Dir("webroot"))
	http.Handle("/", http.StripPrefix("/", fs))
	log.Fatal(http.ListenAndServe(":8080", nil))
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

	event := JsonEvent{Type: 2, Event: string(jsonBytes)}

	events <- event

}
