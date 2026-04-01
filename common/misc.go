package common

import "reflect"

func UNUSED(v ...interface{}) {
}

func ReverseMap(m interface{}) interface{} {
	inputType := reflect.TypeOf(m)
	inputValue := reflect.ValueOf(m)
	result := reflect.MakeMap(reflect.MapOf(inputType.Elem(), inputType.Key()))
	for _, key := range inputValue.MapKeys() {
		result.SetMapIndex(inputValue.MapIndex(key), key)
	}
	return result.Interface()
}
