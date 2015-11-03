package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"

	"golang.org/x/net/websocket"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/hybridgroup/gobot/platforms/intel-iot/edison"
)

var controlLock sync.Mutex
var users bool
var video bool
var videoCmd *exec.Cmd
var wsuser *websocket.Conn = nil

type JsonEvent struct {
	Type  int
	Event string
}

const (
	Signal = 1 << iota
	TrackPower
	Video
	StartVideo
	EndVideo
)

var maxspeed = flag.Int("maxspeed", 255, "max speed")
var maxrotate = flag.Int("maxrotate", 255, "max rotate")

func main() {

	flag.Parse()

	runtime.GOMAXPROCS(2)

	gbot := gobot.NewGobot()

	e := edison.NewEdisonAdaptor("edison")
	process := gpio.NewLedDriver(e, "led", "2")
	connect := gpio.NewLedDriver(e, "led", "4")

	pwmA := gpio.NewDirectPinDriver(e, "pwmA", "3")
	breakA := gpio.NewDirectPinDriver(e, "breakA", "9")
	dirA := gpio.NewDirectPinDriver(e, "dirA", "12")

	pwmB := gpio.NewDirectPinDriver(e, "pwmB", "5")
	breakB := gpio.NewDirectPinDriver(e, "breakB", "8")
	dirB := gpio.NewDirectPinDriver(e, "dirB", "13")

	work := func() {
		tank := NewTank(pwmA, breakA, dirA, pwmB, breakB, dirB, *maxspeed, *maxrotate)
		go runHttpTank(tank, process, connect)
		go runVideo()
	}

	robot := gobot.NewRobot("dartBot",
		[]gobot.Connection{e},
		[]gobot.Device{pwmA},
		[]gobot.Device{breakA},
		[]gobot.Device{dirA},
		[]gobot.Device{pwmB},
		[]gobot.Device{dirB},
		[]gobot.Device{breakB},
		[]gobot.Device{process},
		[]gobot.Device{connect},
		work,
	)

	gbot.AddRobot(robot)
	gbot.Start()

}

func runVideo() {

	ln, err := net.Listen("tcp", "127.0.0.1:8082")
	if err != nil {
		log.Fatal(err.Error())
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("ERROR: Failed to connect clinet: %v\n", err.Error())
			continue
		}

		go handleVideo(conn)
	}

}

func handleVideo(conn net.Conn) {

	buf := make([]byte, 1024)

	for {
		size, err := conn.Read(buf)
		if err != nil {
			log.Printf("ERROR: video recive error: %v\n", err.Error())
			break
		}

		videoToWS(buf[0:size])
	}
}

func videoToWS(data []byte) {

	controlLock.Lock()
	defer controlLock.Unlock()

	if wsuser == nil {
		return
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	event := JsonEvent{Type: Video, Event: encoded}

	if err := websocket.JSON.Send(wsuser, &event); err != nil {
		log.Printf("ERROR: Failed to send video to controler: %v\n", err.Error())
	}

}

func runHttpTank(t *Tank, p *gpio.LedDriver, c *gpio.LedDriver) {

	p.Off()
	c.Off()

	http.Handle("/control", websocket.Handler(func(ws *websocket.Conn) {
		Control(ws, t, p, c)
	}))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getControl(ws *websocket.Conn) bool {
	controlLock.Lock()
	defer controlLock.Unlock()
	if users {
		return false
	}
	users = true
	wsuser = ws
	return true
}

func giveControl() {
	controlLock.Lock()
	defer controlLock.Unlock()
	if users {
		users = false
		wsuser = nil
		endVideoUnsafe()
	}
}

func Control(ws *websocket.Conn, t *Tank, p *gpio.LedDriver, c *gpio.LedDriver) {

	if getControl(ws) {
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
	case StartVideo:
		startVideo()
	case EndVideo:
		endVideo()
	default:
		log.Printf("ERROR: Unknown event (%v) from controler.\n", ev.Type)
	}

}

func startVideo() {
	/*
		controlLock.Lock()
		defer controlLock.Unlock()

		if video {
			return
		}

		go func() {
			//ffmpeg -s 1280x720  -f video4linux2 -i /dev/video0 -f mpeg1video  -r 30 http://127.0.0.1:8082/
			cmd := exec.Command(
				"/home/root/ffmpeg", "-s", "1280x720", "-f", "video4linux2",
				"-i", "/dev/video0", "-f", "mpeg1video",
				"-r", "30", "http://127.0.0.1:8082")

			// VERY VERY DIRTY.
			controlLock.Lock()
			err := cmd.Start()
			videoCmd = cmd
			controlLock.Unlock()

			if err != nil {
				log.Printf("ERROR: Failed to start video encoder: %v.\n", err.Error())
			}

			if err := cmd.Wait(); err != nil {
				log.Printf("ERROR: Video encoder failed: %v.\n", err.Error())
			}

			controlLock.Lock()
			video = false
			controlLock.Unlock()
		}()

		video = true
	*/
}

func endVideo() {
	/*
		controlLock.Lock()
		defer controlLock.Unlock()
		endVideoUnsafe()
	*/
}
func endVideoUnsafe() {

	if !video {
		return
	}

	if videoCmd != nil {
		videoCmd.Process.Kill()
		videoCmd = nil
	}

	video = false
}

func trackPower(t *Tank, js string) {
	var power Power
	if err := json.Unmarshal([]byte(js), &power); err != nil {
		log.Printf("ERROR: Failed to unmsarshal power: %v\n", err.Error())
		return
	}

	t.TrackPower(power)
}
