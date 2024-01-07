package utils

import (
	"fmt"
	"regexp"
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

func DisplayContainerName(name, stack string) string {
	reg := regexp.MustCompile(fmt.Sprintf("/?(%s-)?(?P<name>[a-zA-Z0-9]+(-[0-9]+)?)", stack))
	index := reg.SubexpIndex("name")
	return reg.FindStringSubmatch(name)[index]
}
