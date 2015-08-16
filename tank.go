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

	t.TrackPower(Power{90, 90})

	return t
}

func (t *Tank) TrackPower(p Power) {

	right := (int32(p.Right) - 180) * -1
	p.Right = uint8(right)

	log.Printf("POWER: L(%v) R(%v)\n", p.Left, p.Right)

	t.pinl.PwmWrite(p.Left)
	t.pinr.PwmWrite(p.Right)

}
