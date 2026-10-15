[![Go Reference](https://pkg.go.dev/badge/github.com/VsevolodKamenskiy/go-usbmon.svg)](https://pkg.go.dev/github.com/VsevolodKamenskiy/go-usbmon)

# USBMon

Лёгкая обёртка над libudev для Go, упрощающая отслеживание подключения и отключения USB-устройств. Пакет скрывает сложности низкоуровневого API udev и даёт простой интерфейс, чтобы узнавать о подключении и отключении USB-устройств и реагировать на них.

Пакет сообщает и о подключаемых блочных устройствах: дисках и разделах вроде `/dev/sdb1`. Виртуальные блочные устройства (loop, device-mapper, zram и т. п.) пропускаются.

Это форк [rubiojr/go-usbmon](https://github.com/rubiojr/go-usbmon), адаптированный под Astra Linux 1.7 и Go 1.19.

```Go
// Отслеживание подключения и отключения USB-устройств
package main

import (
	"context"
	"fmt"

	"github.com/VsevolodKamenskiy/go-usbmon"
)

func main() {
	// Печатаем свойства устройства при подключении и отключении
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

Чтобы получать только блочные устройства и узнать, почему мониторинг остановился:

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

Чтобы узнать, какие устройства уже подключены, и к какому USB-устройству относится диск:

```Go
devs, err := usbmon.Connected(&usbmon.SubsystemFilter{Subsystem: "block"})
if err != nil {
	panic(err)
}

for _, dev := range devs {
	// USBDevicePath пуст, если диск подключён не через USB
	fmt.Println(dev.DevNode(), dev.USBDevicePath())
}
```

## Сборка

### Требования

* Go 1.19 или новее
* libudev

Astra Linux, Debian, Ubuntu: `apt install libudev-dev`

Fedora/RHEL: `dnf install systemd-devel`

Arch Linux: `pacman -S systemd-libs`

## Другие примеры

Больше примеров использования — в каталоге [_examples](_examples).

## Лицензия

MIT — подробности в файле [LICENSE](LICENSE).
