package common

import (
	"time"
)

func EncryptTypeX(key []byte, challenge []byte, data []byte) []byte {
	returnData := make([]byte, 20)
	returnData = append(returnData, data...)

	rnd := time.Now().Unix()

	for i := 0; i < 20; i++ {
		rnd = (rnd * 0x343FD) + 0x269EC3
		returnData[i] = byte(rnd ^ int64(key[i%len(key)]) ^ int64(challenge[i%len(challenge)]))
	}

	headerLen := 7
	returnData[0] = byte((headerLen - 2) ^ 0xec)
	returnData[1] = 0x00
	returnData[2] = 0x00
	returnData[headerLen-1] = byte((20 - headerLen) ^ 0xea)

	header := returnData[:20]
	encxkey := make([]byte, 261)
	returnData = initEncrypt(encxkey, key, challenge, returnData)
	func6e(encxkey, returnData)

	return append(header, returnData...)
}

func initEncrypt(encxkey, key, validate, data []byte) []byte {
	// TODO: Bounds
	headerLen := (data[0] ^ 0xec) + 2
	dataStart := (data[headerLen-1] ^ 0xea)

	enctypexFuncX(encxkey, key, validate, data[headerLen:headerLen+dataStart])
	return data[headerLen+dataStart:]
}

func enctypexFuncX(encxkey, key, challenge, data []byte) {
	for i := 0; i < len(data); i++ {
		challenge[(int(key[i%len(key)])*i)&7] ^= challenge[i&7] ^ data[i]
	}

	func4(encxkey, challenge, 8)
}

func func4(encxkey, challenge []byte, challengeLen int) {
	for i := 0; i < 256; i++ {
		encxkey[i] = byte(i)
	}

	n1 := 0
	n2 := 0
	t1 := 0
	for i := 255; i != -1; i-- {
		t1, n1, n2 = func5(encxkey, i, challenge, challengeLen, n1, n2)
		t2 := encxkey[i]
		encxkey[i] = encxkey[t1]
		encxkey[t1] = t2
	}

	encxkey[256] = encxkey[1]
	encxkey[257] = encxkey[3]
	encxkey[258] = encxkey[5]
	encxkey[259] = encxkey[7]
	encxkey[260] = encxkey[n1&0xff]
}

func func5(encxkey []byte, cnt int, id []byte, idLen, n1, n2 int) (int, int, int) {
	if cnt == 0 {
		return 0, n1, n2
	}

	mask := 1
	doLoop := true
	if cnt > 1 {
		for doLoop {
			mask = (mask << 1) + 1
			doLoop = mask < cnt
		}
	}

	i := 0
	tmp := 0
	doLoop = true
	for doLoop {
		n1 = int(encxkey[n1&0xff] + id[n2])
		n2 += 1

		if n2 >= idLen {
			n2 = 0
			n1 += idLen
		}

		tmp = n1 & mask

		i += 1
		if i > 11 {
			tmp %= cnt
		}

		doLoop = tmp > cnt
	}

	return tmp, n1, n2
}

func func6e(encxkey []byte, data []byte) []byte {
	for i := 0; i < len(data); i++ {
		data[i] = func7e(encxkey, data[i])
	}

	return data
}

func func7e(encxkey []byte, d byte) byte {
	a := encxkey[256]
	b := encxkey[257]
	c := encxkey[a]
	encxkey[256] = (a + 1) & 0xff
	encxkey[257] = (b + c) & 0xff

	a = encxkey[260]
	b = encxkey[257]
	b = encxkey[b]
	c = encxkey[a]
	encxkey[a] = b

	a = encxkey[259]
	b = encxkey[257]
	a = encxkey[a]
	encxkey[b] = a

	a = encxkey[256]
	b = encxkey[259]
	a = encxkey[a]
	encxkey[b] = a

	a = encxkey[256]
	encxkey[a] = c

	b = encxkey[258]
	a = encxkey[c]
	c = encxkey[259]
	b = (a + b) & 0xff
	encxkey[258] = b

	a = b
	c = encxkey[c]
	b = encxkey[257]
	b = encxkey[b]
	a = encxkey[a]
	c = (b + c) & 0xff
	b = encxkey[260]
	b = encxkey[b]
	c = (b + c) & 0xff
	b = encxkey[c]
	c = encxkey[256]
	c = encxkey[c]
	a = (a + c) & 0xff
	c = encxkey[b]
	b = encxkey[a]
	c ^= b ^ d
	encxkey[260] = c
	encxkey[259] = d

	return c
}
