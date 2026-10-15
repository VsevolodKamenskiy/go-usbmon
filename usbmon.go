package usbmon

import (
	"context"

	"github.com/jochenvg/go-udev"
)

type Device struct {
	properties map[string]string
}

func newDevice(d *udev.Device) *Device {
	return &Device{properties: d.Properties()}
}

func (d *Device) Serial() string {
	return d.properties["ID_SERIAL_SHORT"]
}

// LongSerial returns ID_SERIAL: the vendor, model and serial number joined with
// underscores, e.g. "Realtek_RTL9210B-CG_012345678943". Unlike Serial, it is
// also set for devices without a serial number, e.g. "1a86_USB_Serial".
func (d *Device) LongSerial() string {
	return d.properties["ID_SERIAL"]
}

func (d *Device) Properties() map[string]string {
	return d.properties
}

func (d *Device) Vendor() string {
	return d.properties["ID_VENDOR"]
}

func (d *Device) Action() string {
	return d.properties["ACTION"]
}

func (d *Device) Major() string {
	return d.properties["MAJOR"]
}

func (d *Device) Minor() string {
	return d.properties["MINOR"]
}

func (d *Device) Path() string {
	return d.properties["DEVPATH"]
}

// Subsystem returns the udev subsystem of the device: "usb" or "block".
func (d *Device) Subsystem() string {
	return d.properties["SUBSYSTEM"]
}

// DevType returns the device type, e.g. "usb_device", "disk" or "partition".
func (d *Device) DevType() string {
	return d.properties["DEVTYPE"]
}

// DevNode returns the device node: e.g. "/dev/sda" or "/dev/sda1" for block
// devices and "/dev/bus/usb/002/003" for USB devices.
func (d *Device) DevNode() string {
	return d.properties["DEVNAME"]
}

func Listen(ctx context.Context) (chan *Device, error) {
	return ListenFiltered(ctx)
}

// ListenFiltered returns the USB devices and the pluggable block devices (disks
// and partitions, but not loop, device-mapper and other virtual ones) that match
// all the filters passed as arguments.
//
// Filters are additive, meaning every device needs to match all the filter arguments.
//
// The returned channel is closed when ctx is done or monitoring fails. Use
// ListenWithErrors to find out why monitoring stopped.
func ListenFiltered(ctx context.Context, filters ...Filter) (chan *Device, error) {
	devs, _, err := listen(ctx, filters)
	return devs, err
}

// ListenWithErrors is like ListenFiltered, but also returns a channel that
// receives the error that stopped monitoring, if any. Both channels are closed
// when monitoring stops.
func ListenWithErrors(ctx context.Context, filters ...Filter) (<-chan *Device, <-chan error, error) {
	return listen(ctx, filters)
}

func listen(ctx context.Context, filters []Filter) (chan *Device, chan error, error) {
	ch, ech, err := NewUdevMonitor().DeviceChan(ctx)
	if err != nil {
		return nil, nil, err
	}

	devs, errs := filterDevices(ctx, ch, ech, newDevice, append([]Filter{pluggable{}}, filters...))
	return devs, errs, nil
}

// filterDevices sends the devices from the go-udev monitor channels ch and ech
// that match all the filters to the returned channel until ctx is done or the
// monitor stops. The error that stopped the monitor, if any, is sent to the
// returned error channel. Both returned channels are closed when filtering stops.
func filterDevices(ctx context.Context, ch <-chan *udev.Device, ech <-chan error, toDevice func(*udev.Device) *Device, filters []Filter) (chan *Device, chan error) {
	devs := make(chan *Device)
	errs := make(chan error, 1)

	go func() {
		defer close(errs)
		defer close(devs)
		defer func() { go drain(ch, ech) }()

		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-ech:
				if ok && err != nil {
					errs <- err
				}
				return
			case d, ok := <-ch:
				if !ok {
					return
				}

				dev := toDevice(d)
				if !matchesAll(dev, filters) {
					continue
				}

				select {
				case devs <- dev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return devs, errs
}

func matchesAll(dev *Device, filters []Filter) bool {
	for _, f := range filters {
		if !f.Matches(dev) {
			return false
		}
	}
	return true
}
