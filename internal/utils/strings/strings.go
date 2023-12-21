package strings

import "errors"

func ReplaceAtIndex(s, v string, i int) (string, error) {
	if len(s) < i {
		return "", errors.New("index can't be greater that string length")
	}
	if len(s) == i {
		return s[:i-1] + v, nil
	}
	return s[:i] + v + s[i+1:], nil
}
