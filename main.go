package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/hybridgroup/gobot/platforms/intel-iot/edison"
)

func main() {
	gbot := gobot.NewGobot()

	e := edison.NewEdisonAdaptor("edison")
	pinl := gpio.NewDirectPinDriver(e, "pin", "3")
	pinr := gpio.NewDirectPinDriver(e, "pin", "5")
	led := gpio.NewLedDriver(e, "led", "2")

	work := func() {
		tank := NewTank(pinl, pinr)
		go runHttpTank(tank, led)
	}

	robot := gobot.NewRobot("dartBot",
		[]gobot.Connection{e},
		[]gobot.Device{pinl},
		[]gobot.Device{pinr},
		[]gobot.Device{led},
		work,
	)

	gbot.AddRobot(robot)
	gbot.Start()

}

func runHttpTank(t *Tank, l *gpio.LedDriver) {

	http.HandleFunc("/power", func(w http.ResponseWriter, r *http.Request) {

		l.Toggle()
		defer l.Toggle()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", 500)
			return
		}

		var power Power
		if err := json.Unmarshal(body, &power); err != nil {
			log.Printf("ERROR: Failed to unmsarshal power: %v\n", err.Error())
			http.Error(w, "Invalid Input", 400)
			return
		}

		t.TrackPower(power)

	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
