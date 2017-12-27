package hm

import "fmt"

type DeviceType int

const (
	DeviceTypeNone DeviceType = iota
	DeviceTypeSwitch
	DeviceTypeBlind
)

type Device struct {
	Type      DeviceType
	HMAddress string
	Room      string
	SetValue  func(v int) error
	Value     func() (int, error)
}

func (d Device) String() string {
	return fmt.Sprintf("%v (%v)", d.HMAddress, d.Room)
}
