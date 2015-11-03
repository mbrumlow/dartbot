package main

import (
	"log"

	"github.com/hybridgroup/gobot/platforms/gpio"
)

type Power struct {
	Left  int16
	Right int16
}

type Tank struct {
	pwmA   *gpio.DirectPinDriver
	breakA *gpio.DirectPinDriver
	dirA   *gpio.DirectPinDriver

	pwmB   *gpio.DirectPinDriver
	breakB *gpio.DirectPinDriver
	dirB   *gpio.DirectPinDriver

	maxspeed  int
	maxrotate int
}

func NewTank(pwmA, breakA, dirA, pwmB, breakB, dirB *gpio.DirectPinDriver, maxspeed int, maxrotate int) *Tank {

	t := &Tank{pwmA: pwmA, breakA: breakA, dirA: dirA, pwmB: pwmB, breakB: breakB, dirB: dirB, maxspeed: maxspeed, maxrotate: maxrotate}

	t.Stop()

	return t
}

func (t *Tank) Stop() {
	t.breakA.DigitalWrite(1)
	t.breakB.DigitalWrite(1)
}

func max(x, y int16) int16 {
	if x > y {
		return x
	}
	return y
}

func min(x, y int16) int16 {
	if x < y {
		return x
	}
	return y
}

func (t *Tank) mapp(m, x int) int {

	a := int(0)
	b := int(255)
	c := int(0)
	d := int(m)

	return (x-a)/(b-a)*(d-c) + c

}

func (t *Tank) TrackPower(p Power) {

	var right uint8
	var left uint8
	var forward = false
	var backwards = false

	log.Printf("POWER: L(%v) R(%v)\n", p.Left, p.Right)

	if p.Right > 0 && p.Left > 0 {
		forward = true
	}

	if p.Right < 0 && p.Left < 0 {
		backwards = true
	}

	if p.Right < 0 {
		p.Right = p.Right * -1
		t.dirA.DigitalWrite(1)
	} else if p.Right > 0 {
		t.dirA.DigitalWrite(0)

	}
	right = uint8(min(255, p.Right))

	if p.Left < 0 {
		p.Left = p.Left * -1
		t.dirB.DigitalWrite(0)
	} else if p.Left > 0 {
		t.dirB.DigitalWrite(1)
	}
	left = uint8(min(255, p.Left))

	log.Printf("PRE REAL POWER: L(%v) R(%v)\n", left, right)

	if forward || backwards {
		right = uint8(t.mapp(t.maxspeed, int(right)))
		left = uint8(t.mapp(t.maxspeed, int(left)))
	} else {
		right = uint8(t.mapp(t.maxrotate, int(right)))
		left = uint8(t.mapp(t.maxrotate, int(left)))
	}

	if right == 0 {
		t.breakA.DigitalWrite(1)
	} else {
		t.breakA.DigitalWrite(0)
	}

	if left == 0 {
		t.breakB.DigitalWrite(1)
	} else {
		t.breakB.DigitalWrite(0)
	}

	log.Printf("REAL POWER: L(%v) R(%v)\n", left, right)

	t.pwmA.PwmWrite(right)
	t.pwmB.PwmWrite(left)

}
