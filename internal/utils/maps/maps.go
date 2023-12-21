package maps

func Keys[Key comparable, Value any](m map[Key]Value) []Key {
	keys := make([]Key, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i++
	}
	return keys
}

func Values[Key comparable, Value any](m map[Key]Value) []Value {
	values := make([]Value, len(m))
	i := 0
	for _, value := range m {
		values[i] = value
		i++
	}
	return values
}
