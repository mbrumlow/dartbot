package main

import (
	"flag"
	"log"
	"time"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/hybridgroup/gobot/platforms/intel-iot/edison"
	"github.com/mbrumlow/dartbot/tank"
	"github.com/mbrumlow/webbot"
)

var host = flag.String("host", "ws://localhost:8080/robot", "Robot Web Control (rwc) host.")
var video = flag.String("video", "/dev/video0", "Path to video device.")
var pass = flag.String("pass", "", "Password to rwc host")

var maxspeed = flag.Int("maxspeed", 255, "max speed")
var maxrotate = flag.Int("maxrotate", 255, "max rotate")

func main() {

	flag.Parse()

	gbot := gobot.NewGobot()

	e := edison.NewEdisonAdaptor("edison")

	pwmA := gpio.NewDirectPinDriver(e, "pwmA", "3")
	breakA := gpio.NewDirectPinDriver(e, "breakA", "9")
	dirA := gpio.NewDirectPinDriver(e, "dirA", "12")

	pwmB := gpio.NewDirectPinDriver(e, "pwmB", "5")
	breakB := gpio.NewDirectPinDriver(e, "breakB", "8")
	dirB := gpio.NewDirectPinDriver(e, "dirB", "13")

	work := func() {
		tank := tank.NewTank(pwmA, breakA, dirA, pwmB, breakB, dirB, *maxspeed, *maxrotate)

		go func() {

			wb := webbot.New(*host, *video, *pass, tank)

			for {
				if err := wb.Run(); err != nil {
					log.Printf("RUN ERROR: %v\n", err.Error())
				}
				time.Sleep(1 * time.Second)
			}

		}()

	}

	robot := gobot.NewRobot("dartBot",
		[]gobot.Connection{e},
		[]gobot.Device{pwmA},
		[]gobot.Device{breakA},
		[]gobot.Device{dirA},
		[]gobot.Device{pwmB},
		[]gobot.Device{dirB},
		[]gobot.Device{breakB},
		work,
	)

	gbot.AddRobot(robot)
	gbot.Start()

}
