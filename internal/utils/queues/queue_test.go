package queues

import (
	"reflect"
	"testing"
)

func TestQueue_PushWithLimit(t *testing.T) {
	queue := New[int]()
	limit := 3

	queue.Push(1)
	err := queue.PushWithLimit(2, limit)
	if err != nil {
		t.Errorf("unexpected error pushing within limit: %v", err)
	}

	err = queue.PushWithLimit(3, limit)
	if err != nil {
		t.Errorf("unexpected error pushing within limit: %v", err)
	}

	if queue.Len() != 3 {
		t.Errorf("unexpected queue length after pushing within limit, got: %v, expected: %v", queue.Len(), 3)
	}

	err = queue.PushWithLimit(4, limit)
	if err != nil {
		t.Errorf("unexpected error pushing within limit: %v", err)
	}

	if queue.Len() != 3 {
		t.Errorf("unexpected queue length after pushing beyond limit, got: %v, expected: %v", queue.Len(), 3)
	}
}

func TestQueue_Len(t *testing.T) {
	queue := New[int]()

	if queue.Len() != 0 {
		t.Errorf("unexpected length of an empty queue, got: %v, expected: %v", queue.Len(), 0)
	}

	queue.Push(1)
	queue.Push(2)
	if queue.Len() != 2 {
		t.Errorf("unexpected length after pushing elements, got: %v, expected: %v", queue.Len(), 2)
	}
}

func TestQueue_Pop(t *testing.T) {
	queue := New[int]()

	_, err := queue.Pop()
	if err == nil {
		t.Errorf("expected error when popping from an empty queue, got: nil")
	}

	queue.Push(1)
	value, err := queue.Pop()
	if err != nil {
		t.Errorf("unexpected error when popping from a non-empty queue, got: %v", err)
	}
	if value != 1 {
		t.Errorf("unexpected value after popping from a non-empty queue, got: %v, expected: %v", value, 1)
	}

	queue.Push(1)
	queue.Push(2)
	value, err = queue.Pop()
	if err != nil {
		t.Errorf("unexpected error when popping from a non-empty queue, got: %v", err)
	}
	if value != 1 {
		t.Errorf("unexpected value after popping from a non-empty queue, got: %v, expected: %v", value, 1)
	}
}

func TestQueue_Last(t *testing.T) {
	queue := New[int]()

	_, err := queue.Last()
	if err == nil {
		t.Errorf("expected error when getting head from an empty queue, got: nil")
	}

	queue.Push(1)
	value, err := queue.Last()
	if err != nil {
		t.Errorf("unexpected error when popping from a non-empty queue, got: %v", err)
	}
	if value != 1 {
		t.Errorf("unexpected value after popping from a non-empty queue, got: %v, expected: %v", value, 1)
	}

	queue.Push(2)
	queue.Push(3)
	value, err = queue.Last()
	if err != nil {
		t.Errorf("unexpected error when getting head from a non-empty queue, got: %v", err)
	}
	if value != 3 {
		t.Errorf("unexpected value as head of a non-empty queue, got: %v, expected: %v", value, 1)
	}

	value, err = queue.Last()
	if err != nil {
		t.Errorf("unexpected error when getting head from a non-empty queue, got: %v", err)
	}
	if value != 3 {
		t.Errorf("unexpected value as head of a non-empty queue, got: %v, expected: %v", value, 1)
	}
}

func TestQueue_ToArray(t *testing.T) {
	queue := New[int]()
	expected := []int{3, 2, 1}

	arr := queue.ToArray()
	if len(arr) != 0 {
		t.Errorf("unexpected length of array for an empty queue, got: %v, expected: %v", len(arr), 0)
	}

	queue.Push(1)
	queue.Push(2)
	queue.Push(3)
	arr = queue.ToArray()
	if !reflect.DeepEqual(arr, expected) {
		t.Errorf("unexpected array after converting a non-empty queue, got: %v, expected: %v", arr, expected)
	}
}
