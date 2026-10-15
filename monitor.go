package usbmon

import (
	"context"

	"github.com/jochenvg/go-udev"
)

type Monitor interface {
	DeviceChan(context.Context) (<-chan *udev.Device, error)
}

type UdevMonitor struct {
	monitor *udev.Monitor
}

func (m *UdevMonitor) DeviceChan(ctx context.Context) (<-chan *udev.Device, <-chan error, error) {
	return m.monitor.DeviceChan(ctx)
}

// NewUdevMonitor returns a monitor of USB devices and block devices.
func NewUdevMonitor() *UdevMonitor {
	u := udev.Udev{}
	m := u.NewMonitorFromNetlink("udev")
	// The usb_device type skips the usb_interface events every device triggers too.
	_ = m.FilterAddMatchSubsystemDevtype("usb", "usb_device")
	_ = m.FilterAddMatchSubsystem("block")

	return &UdevMonitor{monitor: m}
}

// drain discards whatever a go-udev monitor still sends until it closes its
// channels, so its polling goroutine never gets stuck on a send and releases
// its epoll fd.
func drain(devs <-chan *udev.Device, errs <-chan error) {
	for devs != nil || errs != nil {
		select {
		case _, ok := <-devs:
			if !ok {
				devs = nil
			}
		case _, ok := <-errs:
			if !ok {
				errs = nil
			}
		}
	}
}
