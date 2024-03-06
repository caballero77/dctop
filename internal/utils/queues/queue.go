package queues

import (
	"container/list"
	"errors"
	"fmt"
)

type Queue[T any] struct {
	list *list.List
}

func New[T any]() *Queue[T] {
	return &Queue[T]{
		list: list.New(),
	}
}

func (queue *Queue[T]) Push(value T) {
	queue.list.PushBack(value)
}

func (queue *Queue[T]) PushWithLimit(value T, limit int) error {
	queue.Push(value)
	for limit >= 0 && queue.Len() > limit {
		_, err := queue.Pop()
		if err != nil {
			return fmt.Errorf("error getting element from queue: %w", err)
		}
	}
	return nil
}

func (queue Queue[T]) Len() int {
	return queue.list.Len()
}

func (queue *Queue[T]) Pop() (T, error) {
	var value T
	if queue.Len() == 0 {
		return value, errors.New("can't pop value from empty queue")
	}
	value, ok := queue.list.Remove(queue.list.Front()).(T)
	if !ok {
		return value, errors.New("can't convert value from queue")
	}
	return value, nil
}

func (queue Queue[T]) Last() (T, error) {
	var value T
	if queue.Len() == 0 {
		return value, errors.New("can't pop value from empty queue")
	}
	value, ok := queue.list.Back().Value.(T)
	if !ok {
		return value, errors.New("can't convert value from queue")
	}
	return value, nil
}

func (queue Queue[T]) ToArray() []T {
	array := make([]T, queue.Len())
	for e, i := queue.list.Back(), 0; e != nil; e = e.Prev() {
		value, ok := e.Value.(T)
		if ok {
			array[i] = value
		}
		i++
	}
	return array
}
