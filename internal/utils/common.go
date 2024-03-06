package utils

import (
	"fmt"
	"regexp"
)

func DisplayContainerName(name, stack string) string {
	reg := regexp.MustCompile(fmt.Sprintf("/?(%s-)?(?P<name>[a-zA-Z0-9]+(-[0-9]+)?)", stack))
	index := reg.SubexpIndex("name")
	return reg.FindStringSubmatch(name)[index]
}
