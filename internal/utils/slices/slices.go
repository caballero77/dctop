package slices

import (
	"errors"
)

type Number interface {
	byte | int8 | int16 | int | int32 | int64 | float32 | float64
}

func Find[T any](slice []T, predicate func(T) bool) (*T, error) {
	for i := 0; i < len(slice); i++ {
		item := &slice[i]
		if predicate(*item) {
			return item, nil
		}
	}
	var empty T
	return &empty, errors.New("can't find expected item in slice")
}

func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for i := 0; i < len(slice); i++ {
		item := &slice[i]
		if predicate(*item) {
			result = append(result, *item)
		}
	}
	return result
}

func Map[T any, R any](slice []T, functor func(T) (R, error)) ([]R, error) {
	result := make([]R, len(slice))
	for i := 0; i < len(slice); i++ {
		value, err := functor(slice[i])
		if err != nil {
			return nil, err
		}
		result[i] = value
	}
	return result, nil
}

func MapI[T any, R any](slice []T, functor func(int, T) (R, error)) ([]R, error) {
	result := make([]R, len(slice))
	for i := 0; i < len(slice); i++ {
		value, err := functor(i, slice[i])
		if err != nil {
			return nil, err
		}
		result[i] = value
	}
	return result, nil
}

func Sum[T Number](slice []T) T {
	var result T
	for i := 0; i < len(slice); i++ {
		result += slice[i]
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

func Contains[T comparable](slice []T, value T) bool {
	for i := 0; i < len(slice); i++ {
		if slice[i] == value {
			return true
		}
	}
	return false
}

func Remove[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, elem := range slice {
		if predicate(elem) {
			result = append(result, elem)
		}
	}
	return result
}
