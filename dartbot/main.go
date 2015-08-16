package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/hybridgroup/gobot/platforms/intel-iot/edison"
)

var controlLock sync.Mutex
var users int

type JsonEvent struct {
	Type  int
	Event string
}

const (
	Signal = 1 << iota
	TrackPower
)

func main() {
	gbot := gobot.NewGobot()

	e := edison.NewEdisonAdaptor("edison")
	pinl := gpio.NewDirectPinDriver(e, "pin", "3")
	pinr := gpio.NewDirectPinDriver(e, "pin", "5")
	process := gpio.NewLedDriver(e, "led", "2")
	connect := gpio.NewLedDriver(e, "led", "4")

	work := func() {
		tank := NewTank(pinl, pinr)
		go runHttpTank(tank, process, connect)
	}

	robot := gobot.NewRobot("dartBot",
		[]gobot.Connection{e},
		[]gobot.Device{pinl},
		[]gobot.Device{pinr},
		[]gobot.Device{process},
		[]gobot.Device{connect},
		work,
	)

	gbot.AddRobot(robot)
	gbot.Start()

}

func runHttpTank(t *Tank, p *gpio.LedDriver, c *gpio.LedDriver) {

	p.Off()
	c.Off()

	http.Handle("/control", websocket.Handler(func(ws *websocket.Conn) {
		Control(ws, t, p, c)
	}))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getControl() bool {
	controlLock.Lock()
	defer controlLock.Unlock()
	if users > 0 {
		return false
	}
	users++
	return true
}

func giveControl() {
	controlLock.Lock()
	defer controlLock.Unlock()
	if users > 0 {
		users--
	}
}

func Control(ws *websocket.Conn, t *Tank, p *gpio.LedDriver, c *gpio.LedDriver) {

	if getControl() {
		defer giveControl()
	} else {
		log.Printf("Client control denied: %v\n", ws)
		return
	}

	log.Printf("Client taking contro: %v\n", ws)

	c.On()
	defer c.Off()
	defer t.Stop()

	for {

		var ev JsonEvent
		if err := websocket.JSON.Receive(ws, &ev); err != nil {
			log.Printf("Error reciving event: %v\n", err.Error())
			break
		}

		event(ws, t, p, ev)

	}

}

func event(ws *websocket.Conn, t *Tank, p *gpio.LedDriver, ev JsonEvent) {

	p.On()
	defer p.Off()

	switch ev.Type {
	case TrackPower:
		trackPower(t, ev.Event)
	}

}

func trackPower(t *Tank, js string) {
	var power Power
	if err := json.Unmarshal([]byte(js), &power); err != nil {
		log.Printf("ERROR: Failed to unmsarshal power: %v\n", err.Error())
		return
	}

	t.TrackPower(power)
}
