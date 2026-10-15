package usbmon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionFilter(t *testing.T) {
	d := Device{
		properties: map[string]string{"ACTION": "add"},
	}

	f := ActionFilter{Action: ActionAdd}
	assert.True(t, f.Matches(&d))
	d.properties["ACTION"] = "all"
	assert.False(t, f.Matches(&d))
	d.properties["ACTION"] = "remove"
	assert.False(t, f.Matches(&d))

	f = ActionFilter{Action: ActionAll}
	d.properties["ACTION"] = "all"
	assert.True(t, f.Matches(&d))
	d.properties["ACTION"] = "add"
	assert.True(t, f.Matches(&d))
	d.properties["ACTION"] = "remove"
	assert.True(t, f.Matches(&d))
	d.properties["ACTION"] = "unbind"
	assert.False(t, f.Matches(&d))
	d.properties["ACTION"] = "bind"
	assert.False(t, f.Matches(&d))

	f = ActionFilter{Action: ActionRemove}
	d.properties["ACTION"] = "all"
	assert.False(t, f.Matches(&d))
	d.properties["ACTION"] = "add"
	assert.False(t, f.Matches(&d))
	d.properties["ACTION"] = "remove"
	assert.True(t, f.Matches(&d))
}

func TestSerialFilter(t *testing.T) {
	d := Device{
		properties: map[string]string{"ACTION": "add", "ID_SERIAL_SHORT": "1234"},
	}

	f := SerialFilter{Serial: "1234"}
	assert.True(t, f.Matches(&d))
	d.properties["ACTION"] = "bind"
	assert.False(t, f.Matches(&d))
	d.properties["ACTION"] = "remove"
	assert.True(t, f.Matches(&d))
	d.properties["ACTION"] = "unbind"
	assert.False(t, f.Matches(&d))
	f.Serial = ""
	assert.False(t, f.Matches(&d))
	f.Serial = "123"
	assert.False(t, f.Matches(&d))
}

func TestPartitionFilter(t *testing.T) {
	f := PartitionFilter{}
	assert.True(t, f.Matches(&Device{properties: map[string]string{"DEVTYPE": "partition"}}))
	assert.False(t, f.Matches(&Device{properties: map[string]string{"DEVTYPE": "disk"}}))
	assert.False(t, f.Matches(&Device{properties: map[string]string{"DEVTYPE": "usb_device"}}))
}

func TestSubsystemFilter(t *testing.T) {
	f := SubsystemFilter{Subsystem: "block"}
	assert.True(t, f.Matches(&Device{properties: map[string]string{"SUBSYSTEM": "block"}}))
	assert.False(t, f.Matches(&Device{properties: map[string]string{"SUBSYSTEM": "usb"}}))
	assert.False(t, f.Matches(&Device{properties: map[string]string{}}))
}

func TestPluggable(t *testing.T) {
	f := pluggable{}
	for path, want := range map[string]bool{
		"/devices/pci0000:00/0000:00:14.0/usb2/2-1":                                                  true,
		"/devices/pci0000:00/0000:00:14.0/usb2/2-1/2-1:1.0/host0/target0:0:0/0:0:0:0/block/sda/sda1": true,
		"/devices/pci0000:00/0000:00:1d.0/0000:02:00.0/nvme/nvme0/nvme0n1":                           true,
		"/devices/virtual/block/loop0":                                                               false,
		"/devices/virtual/block/dm-0":                                                                false,
		"/devices/virtual/block/zram0":                                                               false,
	} {
		assert.Equal(t, want, f.Matches(&Device{properties: map[string]string{"DEVPATH": path}}), path)
	}
}
