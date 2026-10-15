[![Go Reference](https://pkg.go.dev/badge/github.com/rubiojr/go-usbmon.svg)](https://pkg.go.dev/github.com/rubiojr/go-usbmon)

# USBMon

A lightweight Go wrapper around libudev to simplify monitoring USB device add/remove events. This package abstracts away the complexities of the low-level udev API, providing a simple interface for detecting and responding to USB devices being connected or disconnected.

Pluggable block devices (disks and partitions such as `/dev/sdb1`) are reported too. Virtual block devices (loop, device-mapper, zram, ...) are skipped.

```Go
// monitor USB hotplug events
package main

import (
	"context"
	"fmt"

	"github.com/rubiojr/go-usbmon"
)

func main() {
	// Print device properties when plugged in or unplugged
	filter := &usbmon.ActionFilter{Action: usbmon.ActionAll}
	devs, err := usbmon.ListenFiltered(context.Background(), filter)
	if err != nil {
		panic(err)
	}

	for dev := range devs {
		fmt.Printf("-- Device %s\n", dev.Action())
		fmt.Println("Serial: " + dev.Serial())
		fmt.Println("Path: " + dev.Path())
		fmt.Println("Vendor: " + dev.Vendor())
	}
}
```

To get only block devices and find out why monitoring stopped:

```Go
filters := []usbmon.Filter{
	&usbmon.ActionFilter{Action: usbmon.ActionAll},
	&usbmon.SubsystemFilter{Subsystem: "block"},
}
devs, errs, err := usbmon.ListenWithErrors(ctx, filters...)
if err != nil {
	panic(err)
}

for dev := range devs {
	fmt.Println(dev.Action(), dev.DevType(), dev.DevNode())
}
if err := <-errs; err != nil {
	fmt.Println("monitoring failed:", err)
}
```

## Building

### Requirements

* libudev

Ubuntu/Debian: `apt install libudev-dev`

Fedora/RHEL: `dnf install systemd-devel`

Arch Linux: `pacman -S systemd-libs`

## Additional examples

See [examples](_examples) for more usage examples.

## License

MIT - See [LICENSE](LICENSE) for details.
