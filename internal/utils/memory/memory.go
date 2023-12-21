package memory

import (
	"errors"
	"fmt"
)

const (
	Kilobyte float64 = 1024
	Megabyte         = Kilobyte * 1024
	Gigabyte         = Megabyte * 1024
)

func BytesToReadable(bytes float64) (string, error) {
	if bytes < 0 {
		return "", errors.New("bytes can`t be less than zero")
	}

	if bytes < Kilobyte {
		return fmt.Sprintf("%.2f B", bytes), nil
	}
	if bytes < Megabyte {
		return fmt.Sprintf("%.2f KiB", bytes/Kilobyte), nil
	}
	if bytes < Gigabyte {
		return fmt.Sprintf("%.2f MiB", bytes/Megabyte), nil
	}

	return fmt.Sprintf("%.2f GiB", bytes/Gigabyte), nil
}
