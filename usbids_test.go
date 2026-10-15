package usbmon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const netacPath = "/devices/pci0000:00/0000:00:14.0/usb2/2-2"

// netacEvent returns the properties udev sends for the Netac flash drive: the
// ID_* ones are missing from unbind and remove events.
func netacEvent(action string) *Device {
	props := map[string]string{
		"ACTION":    action,
		"SUBSYSTEM": "usb",
		"DEVTYPE":   "usb_device",
		"DEVPATH":   netacPath,
		"DEVNAME":   "/dev/bus/usb/002/006",
		"PRODUCT":   "dd8/3908/206",
	}
	if action != "unbind" && action != "remove" {
		props["DRIVER"] = "usb"
		props["ID_SERIAL"] = "Netac_DISK_3.2_DD56419884863"
		props["ID_SERIAL_SHORT"] = "DD56419884863"
		props["ID_VENDOR"] = "Netac"
	}
	return &Device{properties: props}
}

func TestUSBIDsFillUnplugEvents(t *testing.T) {
	ids := usbIDs{}
	serial := &SerialFilter{Serial: "DD56419884863"}

	ids.update(netacEvent("add"))
	ids.update(netacEvent("bind"))

	for _, action := range []string{"unbind", "remove"} {
		dev := netacEvent(action)
		ids.update(dev)
		assert.Equal(t, "DD56419884863", dev.Serial(), action)
		assert.Equal(t, "Netac_DISK_3.2_DD56419884863", dev.LongSerial(), action)
		assert.Equal(t, "Netac", dev.Vendor(), action)
		assert.Equal(t, action, dev.Action())
		assert.NotContains(t, dev.Properties(), "DRIVER", "only ID_* properties are filled in")
	}

	remove := netacEvent("remove")
	ids.update(netacEvent("add"))
	ids.update(remove)
	assert.True(t, serial.Matches(remove), "SerialFilter must match the remove event")

	// The device is forgotten once removed.
	assert.Empty(t, ids)
	remove = netacEvent("remove")
	ids.update(remove)
	assert.Empty(t, remove.Serial())
}

func TestUSBIDsKeepEventProperties(t *testing.T) {
	ids := usbIDs{}
	ids.update(netacEvent("add"))

	// A software unbind leaves the device in place, so udev does set its properties.
	unbind := netacEvent("unbind")
	unbind.properties["ID_VENDOR"] = "Netac Technology"
	ids.update(unbind)
	assert.Equal(t, "Netac Technology", unbind.Vendor())
	assert.Equal(t, "DD56419884863", unbind.Serial())
}

func TestUSBIDsAddReplacesDevice(t *testing.T) {
	ids := usbIDs{}
	ids.update(netacEvent("add"))

	// Another device is plugged into the same port, but the remove event of the
	// previous one was missed.
	add := netacEvent("add")
	add.properties["ID_SERIAL"] = "Other_Drive"
	delete(add.properties, "ID_SERIAL_SHORT")
	ids.update(add)

	remove := netacEvent("remove")
	ids.update(remove)
	assert.Equal(t, "Other_Drive", remove.LongSerial())
	assert.Empty(t, remove.Serial())
}

func TestUSBIDsIgnoreBlockDevices(t *testing.T) {
	ids := usbIDs{}
	ids.update(&Device{properties: map[string]string{
		"ACTION":    "add",
		"SUBSYSTEM": "block",
		"DEVPATH":   netacPath + "/2-2:1.0/host1/target1:0:0/1:0:0:0/block/sdb",
		"ID_SERIAL": "Netac_USB_Flash_Drive_YS202010091289HP",
	}})
	assert.Empty(t, ids)
}

func TestUSBIDsRememberConnected(t *testing.T) {
	ids := usbIDs{}
	// Devices found when monitoring starts have no action.
	connected := netacEvent("add")
	delete(connected.properties, "ACTION")
	ids.remember(connected)

	remove := netacEvent("remove")
	ids.update(remove)
	assert.Equal(t, "DD56419884863", remove.Serial())
}
