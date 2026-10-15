package usbmon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jochenvg/go-udev"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeout = time.Second

// fakeDevices maps fake udev devices, which can't be inspected outside of
// go-udev, to the devices they stand for.
type fakeDevices map[*udev.Device]*Device

func (f fakeDevices) add(props map[string]string) *udev.Device {
	d := &udev.Device{}
	f[d] = &Device{properties: props}
	return d
}

func (f fakeDevices) toDevice(d *udev.Device) *Device {
	return f[d]
}

// fakeMonitor behaves like a go-udev monitor goroutine: it sends devs without
// checking ctx, as go-udev does within a batch of events, then waits for ctx
// to be done and closes its channels.
type fakeMonitor struct {
	ch   chan *udev.Device
	ech  chan error
	sent chan struct{} // gets a value once each device has been received
	done chan struct{} // closed once the goroutine has returned
}

func newFakeMonitor(ctx context.Context, devs ...*udev.Device) *fakeMonitor {
	m := &fakeMonitor{
		ch:   make(chan *udev.Device),
		ech:  make(chan error),
		sent: make(chan struct{}, len(devs)),
		done: make(chan struct{}),
	}
	go func() {
		defer close(m.done)
		defer close(m.ch)
		defer close(m.ech)
		for _, d := range devs {
			m.ch <- d
			m.sent <- struct{}{}
		}
		<-ctx.Done()
	}()
	return m
}

// failingMonitor behaves like a go-udev monitor goroutine failing to poll.
func failingMonitor(err error) (ch <-chan *udev.Device, ech <-chan error) {
	c := make(chan *udev.Device)
	e := make(chan error)
	go func() {
		defer close(c)
		defer close(e)
		e <- err
	}()
	return c, e
}

func recvDevice(t *testing.T, devs <-chan *Device) (*Device, bool) {
	t.Helper()
	select {
	case d, ok := <-devs:
		return d, ok
	case <-time.After(timeout):
		t.Fatal("timed out waiting for a device")
		return nil, false
	}
}

func recvErr(t *testing.T, errs <-chan error) error {
	t.Helper()
	select {
	case err := <-errs:
		return err
	case <-time.After(timeout):
		t.Fatal("timed out waiting for an error")
		return nil
	}
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("monitor goroutine did not stop")
	}
}

func TestFilterDevicesMatching(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fakes := fakeDevices{}
	usb := fakes.add(map[string]string{"SUBSYSTEM": "usb"})
	block := fakes.add(map[string]string{"SUBSYSTEM": "block", "DEVNAME": "/dev/sdb"})
	m := newFakeMonitor(ctx, usb, block)
	devs, errs := filterDevices(ctx, m.ch, m.ech, fakes.toDevice, []Filter{&SubsystemFilter{Subsystem: "block"}})

	dev, ok := recvDevice(t, devs)
	require.True(t, ok)
	assert.Equal(t, "/dev/sdb", dev.DevNode())

	cancel()
	_, ok = recvDevice(t, devs)
	assert.False(t, ok)
	assert.NoError(t, recvErr(t, errs))
	waitDone(t, m.done)
}

func TestFilterDevicesError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boom := errors.New("boom")
	ch, ech := failingMonitor(boom)
	devs, errs := filterDevices(ctx, ch, ech, fakeDevices{}.toDevice, nil)

	_, ok := recvDevice(t, devs)
	assert.False(t, ok, "devices channel must be closed on error")
	assert.Equal(t, boom, recvErr(t, errs))
}

func TestFilterDevicesClosedMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan *udev.Device)
	ech := make(chan error)
	close(ch)
	close(ech)

	devs, errs := filterDevices(ctx, ch, ech, fakeDevices{}.toDevice, nil)
	_, ok := recvDevice(t, devs)
	assert.False(t, ok)
	assert.NoError(t, recvErr(t, errs))
}

func TestFilterDevicesCancelWhileNotRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	fakes := fakeDevices{}
	usb1 := fakes.add(map[string]string{"SUBSYSTEM": "usb"})
	usb2 := fakes.add(map[string]string{"SUBSYSTEM": "usb"})
	usb3 := fakes.add(map[string]string{"SUBSYSTEM": "usb"})
	m := newFakeMonitor(ctx, usb1, usb2, usb3)
	devs, errs := filterDevices(ctx, m.ch, m.ech, fakes.toDevice, nil)

	// Nobody reads devs, so once filterDevices has taken the first device it
	// gets stuck sending it, and the monitor gets stuck sending the next one.
	select {
	case <-m.sent:
	case <-time.After(timeout):
		t.Fatal("filterDevices did not take the first device")
	}
	cancel()

	// errs is closed only once filterDevices has returned.
	assert.NoError(t, recvErr(t, errs))
	_, ok := recvDevice(t, devs)
	assert.False(t, ok)
	waitDone(t, m.done)
}

func TestDeviceProperties(t *testing.T) {
	d := Device{properties: map[string]string{
		"SUBSYSTEM":       "block",
		"DEVTYPE":         "partition",
		"DEVNAME":         "/dev/sda1",
		"ID_SERIAL":       "Samsung_SSD_970_EVO_Plus_250GB_S4EUNX0WC17400Z",
		"ID_SERIAL_SHORT": "S4EUNX0WC17400Z",
	}}
	assert.Equal(t, "block", d.Subsystem())
	assert.Equal(t, "partition", d.DevType())
	assert.Equal(t, "/dev/sda1", d.DevNode())
	assert.Equal(t, "Samsung_SSD_970_EVO_Plus_250GB_S4EUNX0WC17400Z", d.LongSerial())
	assert.Equal(t, "S4EUNX0WC17400Z", d.Serial())
}
