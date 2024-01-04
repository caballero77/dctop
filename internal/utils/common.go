package utils

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

func Debounce(duration time.Duration) func(func()) {
	var timer *time.Timer
	var mutex sync.Mutex
	return func(action func()) {
		mutex.Lock()
		defer mutex.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(duration, action)
	}
}

func BeautifyContainerName(name, stack string) string {
	if strings.HasPrefix(name, "/") {
		name = strings.TrimLeft(name, "/")
	}

	stackPrefix := fmt.Sprintf("%s-", stack)
	if strings.HasPrefix(name, stackPrefix) {
		name = strings.TrimLeft(name, stackPrefix)
	}
	return name
}
