// Based on segher's wii.git "ec.c"
// Copyright 2007,2008  Segher Boessenkool  <segher@kernel.crashing.org>

package gpcm

import (
	"bytes"
	"fmt"
	"math/big"
	"wwfc/common"
	"wwfc/logging"
)

var (
	curveN      = new(big.Int).SetBytes([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x13, 0xe9, 0x74, 0xe7, 0x2f, 0x8a, 0x69, 0x22, 0x03, 0x1d, 0x26, 0x03, 0xcf, 0xe0, 0xd7})
	curveGBytes = []byte{0x00, 0xfa, 0xc9, 0xdf, 0xcb, 0xac, 0x83, 0x13, 0xbb, 0x21, 0x39, 0xf1, 0xbb, 0x75, 0x5f, 0xef, 0x65, 0xbc, 0x39, 0x1f, 0x8b, 0x36, 0xf8, 0xf8, 0xeb, 0x73, 0x71, 0xfd, 0x55, 0x8b, 0x01, 0x00, 0x6a, 0x08, 0xa4, 0x19, 0x03, 0x35, 0x06, 0x78, 0xe5, 0x85, 0x28, 0xbe, 0xbf, 0x8a, 0x0b, 0xef, 0xf8, 0x67, 0xa7, 0xca, 0x36, 0x71, 0x6f, 0x7e, 0x01, 0xf8, 0x10, 0x52}
	square      = []byte{0x00, 0x01, 0x04, 0x05, 0x10, 0x11, 0x14, 0x15, 0x40, 0x41, 0x44, 0x45, 0x50, 0x51, 0x54, 0x55}
)

func eltIsZero(d []byte) bool {
	for i := 0; i < 30; i++ {
		if d[i] != 0 {
			return false
		}
	}
	return true
}

func eltAdd(a []byte, b []byte) []byte {
	d := make([]byte, 30)
	for i := 0; i < 30; i++ {
		d[i] = a[i] ^ b[i]
	}
	return d
}

func eltMulX(d []byte, a []byte) {
	carry := a[0] & 1

	x := byte(0)
	for i := 0; i < 29; i++ {
		y := a[i+1]
		d[i] = x ^ (y >> 7)
		x = y << 1
	}
	d[29] = x ^ carry
	d[20] ^= carry << 2
}

func eltMul(a []byte, b []byte) []byte {
	d := make([]byte, 30)

	i := 0
	mask := byte(1)
	for n := 0; n < 233; n++ {
		eltMulX(d, d)

		if (a[i] & mask) != 0 {
			d = eltAdd(d, b)
		}

		mask >>= 1
		if mask == 0 {
			mask = 0x80
			i++
		}
	}

	return d
}

func eltSquareToWide(a []byte) []byte {
	d := make([]byte, 60)
	for i := 0; i < 30; i++ {
		d[2*i] = square[a[i]>>4]
		d[2*i+1] = square[a[i]&15]
	}
	return d
}

func wideReduce(d []byte) {
	for i := 0; i < 30; i++ {
		x := d[i]

		d[i+19] ^= x >> 7
		d[i+20] ^= x << 1

		d[i+29] ^= x >> 1
		d[i+30] ^= x << 7
	}

	x := d[30] & 0xfe

	d[49] ^= x >> 7
	d[50] ^= x << 1

	d[59] ^= x >> 1

	d[30] &= 1
}

func eltSquare(a []byte) []byte {
	wide := eltSquareToWide(a)
	wideReduce(wide)
	return wide[30:]
}

func itohTsujii(a []byte, b []byte, j uint32) []byte {
	t := bytes.Clone(a)
	for ; j != 0; j-- {
		t = eltSquare(t)
	}

	return eltMul(t, b)
}

func eltInv(a []byte) []byte {

	t := itohTsujii(a, a, 1)
	s := itohTsujii(t, a, 1)
	t = itohTsujii(s, s, 3)
	s = itohTsujii(t, a, 1)
	t = itohTsujii(s, s, 7)
	s = itohTsujii(t, t, 14)
	t = itohTsujii(s, a, 1)
	s = itohTsujii(t, t, 29)
	t = itohTsujii(s, s, 58)
	s = itohTsujii(t, t, 116)
	return eltSquare(s)
}

func pointIsZero(p []byte) bool {
	for i := 0; i < 60; i++ {
		if p[i] != 0 {
			return false
		}
	}
	return true
}

func pointDouble(p []byte) []byte {
	px := p[:30]
	py := p[30:]

	if eltIsZero(px) {
		return make([]byte, 60)
	}

	t := eltInv(px)
	s := eltMul(py, t)
	s = eltAdd(s, px)

	t = eltSquare(px)

	rx := eltSquare(s)
	rx = eltAdd(rx, s)
	rx[29] ^= 1

	ry := eltMul(s, rx)
	ry = eltAdd(ry, rx)
	ry = eltAdd(ry, t)

	return append(rx, ry...)
}

func pointAdd(p []byte, q []byte) []byte {
	if pointIsZero(p) {
		return bytes.Clone(q)
	}

	if pointIsZero(q) {
		return bytes.Clone(p)
	}

	px := p[:30]
	py := p[30:]
	qx := q[:30]
	qy := q[30:]

	u := eltAdd(px, qx)

	if eltIsZero(u) {
		u = eltAdd(py, qy)
		if eltIsZero(u) {
			return pointDouble(p)
		}
		return make([]byte, 60)
	}

	t := eltInv(u)
	u = eltAdd(py, qy)
	s := eltMul(t, u)

	t = eltSquare(s)
	t = eltAdd(t, s)
	t = eltAdd(t, qx)
	t[29] ^= 1

	u = eltMul(s, t)
	s = eltAdd(u, py)
	rx := eltAdd(t, px)
	ry := eltAdd(s, rx)

	return append(rx, ry...)
}

func pointMul(a []byte, b []byte) []byte {
	d := make([]byte, 60)

	for i := 0; i < 30; i++ {
		for mask := byte(0x80); mask != 0; mask >>= 1 {
			d = pointDouble(d)
			if (a[i] & mask) != 0 {
				d = pointAdd(d, b)
			}
		}
	}
	return d
}

func bigIntToBytes(a *big.Int) []byte {
	r := a.Bytes()
	if len(r) < 30 {
		return append(make([]byte, 30-len(r)), r...)
	}
	return r
}

func printHex(data []byte) {
	logMsg := ""
	for i := 0; i < len(data); i++ {
		if (i % 16) == 0 {
			logMsg += "\n"
		}
		logMsg += fmt.Sprintf("%02x ", data[i])
	}
	logging.Info("GPCM", "Data:", logMsg)
}

func verifyECDSA(publicKey []byte, signature []byte, hash []byte) bool {
	common.UNUSED(printHex)

	r := big.NewInt(0).SetBytes(signature[0x00:0x1E])
	s := big.NewInt(0).SetBytes(signature[0x1E:0x3C])

	inv := big.NewInt(0).ModInverse(s, curveN)
	e := big.NewInt(0).SetBytes(hash)

	// printHex(bigIntToBytes(inv))

	w1 := big.NewInt(0).Mul(e, inv)
	w1.Mod(w1, curveN)
	w2 := big.NewInt(0).Mul(r, inv)
	w2.Mod(w2, curveN)

	// printHex(bigIntToBytes(w1))
	// printHex(bigIntToBytes(w2))

	r1 := pointMul(bigIntToBytes(w1), curveGBytes)
	r2 := pointMul(bigIntToBytes(w2), publicKey)
	r3 := pointAdd(r1, r2)
	rx := big.NewInt(0).SetBytes(r3[:30])

	// printHex(r1)
	// printHex(r2)
	// printHex(bigIntToBytes(rx))

	if rx.Cmp(curveN) >= 0 {
		// TODO: This is correct right?
		rx.Sub(rx, curveN)
		rx.Mod(rx, curveN)
	}

	return rx.Cmp(r) == 0
}
