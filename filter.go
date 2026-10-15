package usbmon

import "strings"

type Filter interface {
	Matches(*Device) bool
}

type ActionEvent string

const (
	ActionAdd    ActionEvent = "add"
	ActionRemove ActionEvent = "remove"
	ActionAll    ActionEvent = "all"
)

type ActionFilter struct {
	Action ActionEvent
}

func (f *ActionFilter) Matches(dev *Device) bool {
	action := ActionEvent(dev.Properties()["ACTION"])

	if f.Action == ActionAll && (action == ActionAdd || action == ActionRemove) {
		return true
	}

	return action == f.Action
}

type SerialFilter struct {
	Serial string
}

func (f *SerialFilter) Matches(dev *Device) bool {
	action := ActionEvent(dev.Action())

	return f.Serial == dev.Serial() && (action == ActionAdd || action == ActionRemove)
}

type PartitionFilter struct {
	Serial string
}

func (f *PartitionFilter) Matches(dev *Device) bool {
	return dev.DevType() == "partition"
}

// SubsystemFilter matches the devices of the given udev subsystem: "usb" or "block".
type SubsystemFilter struct {
	Subsystem string
}

func (f *SubsystemFilter) Matches(dev *Device) bool {
	return dev.Subsystem() == f.Subsystem
}

// pluggable drops virtual devices, such as loop, device-mapper or zram block
// devices, as they are not backed by hardware that can be plugged in.
type pluggable struct{}

func (pluggable) Matches(dev *Device) bool {
	return !strings.HasPrefix(dev.Path(), "/devices/virtual/")
}
