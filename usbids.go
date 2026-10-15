package usbmon

import (
	"errors"
	"strings"

	"github.com/jochenvg/go-udev"
)

// usbIDs keeps the ID_* properties (ID_SERIAL, ID_VENDOR, ...) of the connected
// USB devices by DEVPATH.
//
// When a USB device is unplugged, udev handles its unbind event once the device
// is already gone from sysfs, so udev rules can't read the device and the ID_*
// properties go missing from the unbind and remove events.
type usbIDs map[string]map[string]string

// connectedUSBIDs returns the ID_* properties of the USB devices connected
// before monitoring starts, so that their unbind and remove events get them too.
func connectedUSBIDs() (usbIDs, error) {
	u := udev.Udev{}
	e := u.NewEnumerate()
	if e == nil {
		return nil, errors.New("udev: udev_enumerate_new failed")
	}
	if err := e.AddMatchSubsystem("usb"); err != nil {
		return nil, err
	}
	devs, err := e.Devices()
	if err != nil {
		return nil, err
	}

	ids := usbIDs{}
	for _, d := range devs {
		// d is nil if the device was unplugged while enumerating.
		if d == nil {
			continue
		}
		if dev := newDevice(d); dev.DevType() == "usb_device" {
			ids.remember(dev)
		}
	}
	return ids, nil
}

// update records the ID_* properties of USB devices from their events and fills
// them in to the unbind and remove events that lack them.
func (ids usbIDs) update(dev *Device) {
	if dev.Subsystem() != "usb" {
		return
	}

	path := dev.Path()
	switch dev.Action() {
	case "unbind", "remove":
		for k, v := range ids[path] {
			if _, ok := dev.properties[k]; !ok {
				dev.properties[k] = v
			}
		}
		if dev.Action() == "remove" {
			delete(ids, path)
		}
	case "add":
		// Forget a device on the same port whose remove event was missed.
		delete(ids, path)
		ids.remember(dev)
	default:
		ids.remember(dev)
	}
}

// remember records the ID_* properties of dev.
func (ids usbIDs) remember(dev *Device) {
	path := dev.Path()
	for k, v := range dev.properties {
		if !strings.HasPrefix(k, "ID_") {
			continue
		}
		if ids[path] == nil {
			ids[path] = map[string]string{}
		}
		ids[path][k] = v
	}
}
