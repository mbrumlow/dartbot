package main

import (
	"log"

	"github.com/hybridgroup/gobot/platforms/gpio"
)

type Power struct {
	Left  uint8
	Right uint8
}

type Tank struct {
	pinl *gpio.DirectPinDriver
	pinr *gpio.DirectPinDriver
}

func NewTank(pinl, pinr *gpio.DirectPinDriver) *Tank {

	t := &Tank{pinl: pinl, pinr: pinr}

	t.Stop()

	return t
}

func (t *Tank) Stop() {
	// TODO - see if ther is a way (there likely is) to stop sending PWM
	t.TrackPower(Power{90, 90})
}

func (t *Tank) TrackPower(p Power) {

	right := (int32(p.Right) - 180) * -1
	p.Right = uint8(right)

	log.Printf("POWER: L(%v) R(%v)\n", p.Left, p.Right)

	t.pinl.PwmWrite(p.Left)
	t.pinr.PwmWrite(p.Right)

}
