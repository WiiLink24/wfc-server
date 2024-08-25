package api

// make map string string
func mmss(data ...string) map[string]string {
	ret := make(map[string]string)

	l := len(data)

	if l%2 != 0 || l == 0 {
		panic("Length of data must be divisible by two")
	}

	for i := 0; i < l; i += 2 {
		ret[data[i]] = data[i+1]
	}

	return ret
}
