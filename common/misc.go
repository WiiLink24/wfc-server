package common

import "reflect"

// MaybeUnused is a helper function to mark variables as used to avoid compiler warnings.
func MaybeUnused(v ...interface{}) struct{} {
	return struct{}{}
}

// ShouldNotError panics if the error is not nil.
func ShouldNotError(err error) {
	if err != nil {
		panic(err)
	}
}

// ReverseMap takes a map and returns a new map with the keys and values reversed.
func ReverseMap(m interface{}) interface{} {
	inputType := reflect.TypeOf(m)
	inputValue := reflect.ValueOf(m)
	result := reflect.MakeMap(reflect.MapOf(inputType.Elem(), inputType.Key()))
	for _, key := range inputValue.MapKeys() {
		result.SetMapIndex(inputValue.MapIndex(key), key)
	}
	return result.Interface()
}
