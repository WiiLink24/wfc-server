package common

// CalculateMiiCRC
// https://github.com/kiwi515/ogws/blob/ee72b2a329c50b773fcdce5c302e5674e5f2cf11/src/RVLFaceLib/RFL_Database.c#L640
func CalculateMiiCRC(data []byte) uint16 {
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
