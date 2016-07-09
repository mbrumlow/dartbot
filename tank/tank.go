package tank

import (
	"log"

	"github.com/hybridgroup/gobot/platforms/gpio"
)

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

func NewTank(pwmA, breakA, dirA, pwmB, breakB, dirB *gpio.DirectPinDriver,
	maxspeed int, maxrotate int) *Tank {

	t := &Tank{
		pwmA:      pwmA,
		breakA:    breakA,
		dirA:      dirA,
		pwmB:      pwmB,
		breakB:    breakB,
		dirB:      dirB,
		maxspeed:  maxspeed,
		maxrotate: maxrotate}

	t.Stop()

	return t
}

func (t *Tank) Stop() error {

	log.Printf("Stop.\n")

	t.breakA.DigitalWrite(1)
	t.breakB.DigitalWrite(1)

	return nil
}

func (t *Tank) Right() error {

	log.Printf("Right.\n")

	t.breakA.DigitalWrite(0)
	t.breakB.DigitalWrite(0)

	t.dirB.DigitalWrite(0)
	t.dirA.DigitalWrite(0)

	t.pwmA.PwmWrite(uint8(t.maxrotate))
	t.pwmB.PwmWrite(uint8(t.maxrotate))

	return nil
}

func (t *Tank) Left() error {

	log.Printf("Left.\n")

	t.breakA.DigitalWrite(0)
	t.breakB.DigitalWrite(0)

	t.dirB.DigitalWrite(1)
	t.dirA.DigitalWrite(1)

	t.pwmA.PwmWrite(uint8(t.maxrotate))
	t.pwmB.PwmWrite(uint8(t.maxrotate))

	return nil
}

func (t *Tank) Forward() error {

	log.Printf("Forward.\n")

	t.breakA.DigitalWrite(0)
	t.breakB.DigitalWrite(0)

	t.dirB.DigitalWrite(0)
	t.dirA.DigitalWrite(1)

	t.pwmA.PwmWrite(uint8(t.maxspeed))
	t.pwmB.PwmWrite(uint8(t.maxspeed))

	return nil
}

func (t *Tank) Backward() error {

	log.Printf("Backward.\n")

	t.breakA.DigitalWrite(0)
	t.breakB.DigitalWrite(0)

	t.dirB.DigitalWrite(1)
	t.dirA.DigitalWrite(0)

	t.pwmA.PwmWrite(uint8(t.maxspeed))
	t.pwmB.PwmWrite(uint8(t.maxspeed))

	return nil
}
