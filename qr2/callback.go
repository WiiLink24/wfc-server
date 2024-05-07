package qr2

var gpErrorCallback func(uint32, string)

func SetGPErrorCallback(callback func(uint32, string)) {
	gpErrorCallback = callback
}
