package utils

import (
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
