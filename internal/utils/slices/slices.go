package slices

import (
	"errors"
)

type Number interface {
	byte | int8 | int16 | int | int32 | int64 | float32 | float64
}

func Find[T any](array []T, predicate func(T) bool) (*T, error) {
	for i := 0; i < len(array); i++ {
		item := &array[i]
		if predicate(*item) {
			return item, nil
		}
	}
	var empty T
	return &empty, errors.New("can't find expected item in array")
}

func Filter[T any](array []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for i := 0; i < len(array); i++ {
		item := &array[i]
		if predicate(*item) {
			result = append(result, *item)
		}
	}
	return result
}

func Map[T any, R any](array []T, functor func(T) (R, error)) ([]R, error) {
	result := make([]R, len(array))
	for i := 0; i < len(array); i++ {
		value, err := functor(array[i])
		if err != nil {
			return nil, err
		}
		result[i] = value
	}
	return result, nil
}

func MapI[T any, R any](array []T, functor func(int, T) (R, error)) ([]R, error) {
	result := make([]R, len(array))
	for i := 0; i < len(array); i++ {
		value, err := functor(i, array[i])
		if err != nil {
			return nil, err
		}
		result[i] = value
	}
	return result, nil
}

func Sum[T Number](array []T) T {
	var result T
	for i := 0; i < len(array); i++ {
		result += array[i]
	}
	return result
}

func Repeat[T any](value T, size int) []T {
	result := make([]T, size)
	for i := 0; i < size; i++ {
		result[i] = value
	}
	return result
}
