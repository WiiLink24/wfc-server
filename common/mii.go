package common

// References:
// https://wiibrew.org/wiki/Mii_Data
// https://github.com/kiwi515/ogws/tree/master/src/RVLFaceLib

type Mii [0x4C]byte

func (data Mii) RFLCalculateCRC() uint16 {
	crc := uint16(0)

	for _, val := range data {
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc <<= 1
				crc ^= 0x1021
			} else {
				crc <<= 1
			}

			if val&0x80 != 0 {
				crc ^= 0x1
			}

			val <<= 1
		}
	}

	return crc
}

func (data Mii) GetPlayerName() string {
	return string(data[0x02:0x15])
}

var officialMiiList = []uint64{
	0x80000000ECFF82D2,
	0x80000001ECFF82D2,
	0x80000002ECFF82D2,
	0x80000003ECFF82D2,
	0x80000004ECFF82D2,
	0x80000005ECFF82D2,
}

func RFLSearchOfficialData(id uint64) (bool, int) {
	for i, v := range officialMiiList {
		if v == id {
			return true, i
		}
	}

	return false, -1
}
