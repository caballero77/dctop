package plotting

import "golang.org/x/exp/constraints"

type PushMsg[T constraints.Float] struct {
	Value T
}
