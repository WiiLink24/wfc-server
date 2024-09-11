package common

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type MarioKartWiiLeaderboardRegionId int
type MarioKartWiiCourseId int
type MarioKartWiiCharacterId int
type MarioKartWiiVehicleId int
type MarioKartWiiControllerId int
type MarioKartWiiWeightClassId int

// MarioKartWiiLeaderboardRegionId
const (
	Worldwide    = iota // 0x00
	Japan               // 0x01
	UnitedStates        // 0x02
	Europe              // 0x03
	Australia           // 0x04
	Taiwan              // 0x05
	Korea               // 0x06
	China               // 0x07
)

// MarioKartWiiCourseId
const (
	MarioCircuit        = iota // 0x00
	MooMooMeadows              // 0x01
	MushroomGorge              // 0x02
	GrumbleVolcano             // 0x03
	ToadsFactory               // 0x04
	CoconutMall                // 0x05
	DKSummit                   // 0x06
	WarioGoldMine              // 0x07
	LuigiCircuit               // 0x08
	DaisyCircuit               // 0x09
	MoonviewHighway            // 0x0A
	MapleTreeway               // 0x0B
	BowsersCastle              // 0x0C
	RainbowRoad                // 0x0D
	DryDryRuins                // 0x0E
	KoopaCape                  // 0x0F
	GCNPeachBeach              // 0x10
	GCNMarioCircuit            // 0x11
	GCNWaluigiStadium          // 0x12
	GCNDKMountain              // 0x13
	DSYoshiFalls               // 0x14
	DSDesertHills              // 0x15
	DSPeachGardens             // 0x16
	DSDelfinoSquare            // 0x17
	SNESMarioCircuit3          // 0x18
	SNESGhostValley2           // 0x19
	N64MarioRaceway            // 0x1A
	N64SherbetLand             // 0x1B
	N64BowsersCastle           // 0x1C
	N64DKsJungleParkway        // 0x1D
	GBABowserCastle3           // 0x1E
	GBAShyGuyBeach             // 0x1F
)

// MarioKartWiiCharacterId
const (
	Mario                  = iota // 0x00
	BabyPeach                     // 0x01
	Waluigi                       // 0x02
	Bowser                        // 0x03
	BabyDaisy                     // 0x04
	DryBones                      // 0x05
	BabyMario                     // 0x06
	Luigi                         // 0x07
	Toad                          // 0x08
	DonkeyKong                    // 0x09
	Yoshi                         // 0x0A
	Wario                         // 0x0B
	BabyLuigi                     // 0x0C
	Toadette                      // 0x0D
	KoopaTroopa                   // 0x0E
	Daisy                         // 0x0F
	Peach                         // 0x10
	Birdo                         // 0x11
	DiddyKong                     // 0x12
	KingBoo                       // 0x13
	BowserJr                      // 0x14
	DryBowser                     // 0x15
	FunkyKong                     // 0x16
	Rosalina                      // 0x17
	SmallMiiOutfitAMale           // 0x18
	SmallMiiOutfitAFemale         // 0x19
	SmallMiiOutfitBMale           // 0x1A
	SmallMiiOutfitBFemale         // 0x1B
	SmallMiiOutfitCMale           // 0x1C
	SmallMiiOutfitCFemale         // 0x1D
	MediumMiiOutfitAMale          // 0x1E
	MediumMiiOutfitAFemale        // 0x1F
	MediumMiiOutfitBMale          // 0x20
	MediumMiiOutfitBFemale        // 0x21
	MediumMiiOutfitCMale          // 0x22
	MediumMiiOutfitCFemale        // 0x23
	LargeMiiOutfitAMale           // 0x24
	LargeMiiOutfitAFemale         // 0x25
	LargeMiiOutfitBMale           // 0x26
	LargeMiiOutfitBFemale         // 0x27
	LargeMiiOutfitCMale           // 0x28
	LargeMiiOutfitCFemale         // 0x29
)

// MarioKartWiiVehicleId
const (
	StandardKartSmall  = iota // 0x00
	StandardKartMedium        // 0x01
	StandardKartLarge         // 0x02
	BoosterSeat               // 0x03
	ClassicDragster           // 0x04
	Offroader                 // 0x05
	MiniBeast                 // 0x06
	WildWing                  // 0x07
	FlameFlyer                // 0x08
	CheepCharger              // 0x09
	SuperBlooper              // 0x0A
	PiranhaProwler            // 0x0B
	TinyTitan                 // 0x0C
	Daytripper                // 0x0D
	Jetsetter                 // 0x0E
	BlueFalcon                // 0x0F
	Sprinter                  // 0x10
	Honeycoupe                // 0x11
	StandardBikeSmall         // 0x12
	StandardBikeMedium        // 0x13
	StandardBikeLarge         // 0x14
	BulletBike                // 0x15
	MachBike                  // 0x16
	FlameRunner               // 0x17
	BitBike                   // 0x18
	Sugarscoot                // 0x19
	WarioBike                 // 0x1A
	Quacker                   // 0x1B
	ZipZip                    // 0x1C
	ShootingStar              // 0x1D
	Magikruiser               // 0x1E
	Sneakster                 // 0x1F
	Spear                     // 0x20
	JetBubble                 // 0x21
	DolphinDasher             // 0x22
	Phantom                   // 0x23
)

// MarioKartWiiControllerId
const (
	WiiWheel             = iota // 0x00
	WiiRemoteAndNunchuck        // 0x01
	Classic                     // 0x02
	GameCube                    // 0x03
)

// MarioKartWiiWeightClassId
const (
	LightWeight = iota
	MiddleWeight
	HeavyWeight
)

func (regionId MarioKartWiiLeaderboardRegionId) IsValid() bool {
	return regionId >= Worldwide && regionId <= China
}

func (courseId MarioKartWiiCourseId) IsValid() bool {
	return courseId >= MarioCircuit && courseId <= GBAShyGuyBeach
}

func (characterId MarioKartWiiCharacterId) IsValid() bool {
	// Mii Outfit C is not allowed
	if characterId == SmallMiiOutfitCMale || characterId == SmallMiiOutfitCFemale || characterId == MediumMiiOutfitCMale || characterId == MediumMiiOutfitCFemale {
		return false
	}

	return characterId >= Mario && characterId <= LargeMiiOutfitBFemale
}

func (characterId MarioKartWiiCharacterId) GetWeightClass() MarioKartWiiWeightClassId {
	switch characterId {
	case BabyPeach, BabyDaisy, DryBones, BabyMario, Toad, BabyLuigi, Toadette, KoopaTroopa, SmallMiiOutfitAMale, SmallMiiOutfitAFemale, SmallMiiOutfitBMale, SmallMiiOutfitBFemale, SmallMiiOutfitCMale, SmallMiiOutfitCFemale:
		return LightWeight

	case Mario, Luigi, Yoshi, Daisy, Peach, Birdo, DiddyKong, BowserJr, MediumMiiOutfitAMale, MediumMiiOutfitAFemale, MediumMiiOutfitBMale, MediumMiiOutfitBFemale, MediumMiiOutfitCMale, MediumMiiOutfitCFemale:
		return MiddleWeight

	case Waluigi, Bowser, Wario, DonkeyKong, KingBoo, DryBowser, FunkyKong, Rosalina, LargeMiiOutfitAMale, LargeMiiOutfitAFemale, LargeMiiOutfitBMale, LargeMiiOutfitBFemale, LargeMiiOutfitCMale, LargeMiiOutfitCFemale:
		return HeavyWeight
	}

	return -1
}

func (vehicleId MarioKartWiiVehicleId) IsValid() bool {
	return vehicleId >= StandardKartSmall && vehicleId <= Phantom
}

func (vehicleId MarioKartWiiVehicleId) GetWeightClass() MarioKartWiiWeightClassId {
	if vehicleId < 0 {
		return -1
	}

	return MarioKartWiiWeightClassId(vehicleId % 3)
}

func (controllerId MarioKartWiiControllerId) IsValid() bool {
	return controllerId >= WiiWheel && controllerId <= GameCube
}

type RKGhostData []byte

const (
	RKGDFileMaxSize = 0x2800
	RKGDFileMinSize = 0x0088 + 0x0014 + 0x0004
)

func (rkgd RKGhostData) GetBits(byteOffset int, bitOffset int, bitLength int) uint32 {
	if bitLength == 0 {
		return 0
	}

	byteIndex := byteOffset + (bitOffset / 8)
	bitIndex := bitOffset % 8
	byteCount := (bitLength + bitIndex + 7) / 8

	var value uint32
	for i := 0; i < byteCount; i++ {
		value |= uint32(rkgd[byteIndex+i]) << uint32((byteCount-i-1)*8)
	}

	endBitIndex := (bitIndex + bitLength) % 8
	value >>= uint32(8-endBitIndex) % 8
	value &= (1 << uint32(bitLength)) - 1

	return value
}

func (rkgd RKGhostData) GetMinutes(lap int) int {
	if lap == 0 {
		return int(rkgd.GetBits(0x04, 0, 7))
	}

	if lap < 1 || lap > 5 {
		return 0
	}

	return int(rkgd.GetBits(0x11+(lap-1)*3, 0, 7))
}

func (rkgd RKGhostData) GetSeconds(lap int) int {
	if lap == 0 {
		return int(rkgd.GetBits(0x04, 7, 7))
	}

	if lap < 1 || lap > 5 {
		return 0
	}

	return int(rkgd.GetBits(0x11+(lap-1)*3, 7, 7))
}

func (rkgd RKGhostData) GetMilliseconds(lap int) int {
	if lap == 0 {
		return int(rkgd.GetBits(0x05, 6, 10))
	}

	if lap < 1 || lap > 5 {
		return 0
	}

	return int(rkgd.GetBits(0x11+(lap-1)*3, 14, 10))
}

func (rkgd RKGhostData) GetTime(lap int) int {
	return rkgd.GetMinutes(lap)*60000 + rkgd.GetSeconds(lap)*1000 + rkgd.GetMilliseconds(lap)
}

func (rkgd RKGhostData) GetCourse() MarioKartWiiCourseId {
	return MarioKartWiiCourseId(rkgd.GetBits(0x07, 0, 6))
}

func (rkgd RKGhostData) GetVehicle() MarioKartWiiVehicleId {
	return MarioKartWiiVehicleId(rkgd.GetBits(0x08, 0, 6))
}

func (rkgd RKGhostData) GetCharacter() MarioKartWiiCharacterId {
	return MarioKartWiiCharacterId(rkgd.GetBits(0x08, 6, 6))
}

func (rkgd RKGhostData) GetYear() int {
	return int(rkgd.GetBits(0x09, 4, 7))
}

func (rkgd RKGhostData) GetMonth() int {
	return int(rkgd.GetBits(0x0A, 3, 4))
}

func (rkgd RKGhostData) GetDay() int {
	return int(rkgd.GetBits(0x0A, 7, 5))
}

func (rkgd RKGhostData) GetController() MarioKartWiiControllerId {
	return MarioKartWiiControllerId(rkgd.GetBits(0x0B, 4, 4))
}

func (rkgd RKGhostData) IsCompressed() bool {
	return rkgd.GetBits(0x0C, 4, 1) == 1
}

func (rkgd RKGhostData) GetGhostType() int {
	return int(rkgd.GetBits(0x0C, 7, 7))
}

func (rkgd RKGhostData) GetDriftType() int {
	return int(rkgd.GetBits(0x0D, 6, 1))
}

func (rkgd RKGhostData) GetInputDataLength() uint16 {
	return uint16(rkgd.GetBits(0x0E, 0, 16))
}

func (rkgd RKGhostData) GetLapCount() int {
	return int(rkgd.GetBits(0x10, 0, 8))
}

func (rkgd RKGhostData) GetCountryCode() byte {
	return byte(rkgd.GetBits(0x34, 0, 8))
}

func (rkgd RKGhostData) GetStateCode() byte {
	return byte(rkgd.GetBits(0x35, 0, 8))
}

func (rkgd RKGhostData) GetLocationCode() uint16 {
	return uint16(rkgd.GetBits(0x36, 0, 16))
}

func (rkgd RKGhostData) GetMiiData() Mii {
	return Mii(rkgd[0x3C : 0x3C+0x4C])
}

func (rkgd RKGhostData) GetCompressedSize() uint32 {
	return binary.BigEndian.Uint32(rkgd[0x88:0x8C])
}

func (rkgd RKGhostData) GetCompressedData() []byte {
	return []byte(rkgd[0x8C : len(rkgd)-4])
}

func (rkgd RKGhostData) IsRKGDFileValid(moduleName string, expectedCourse MarioKartWiiCourseId, expectedScore int) bool {
	rkgdFileMagic := []byte{'R', 'K', 'G', 'D'}
	rkgdFileLength := len(rkgd)

	if rkgdFileLength < RKGDFileMinSize || rkgdFileLength > RKGDFileMaxSize {
		logging.Error(moduleName, "Invalid RKGD length:", aurora.Cyan(rkgdFileLength))
		return false
	}

	if !bytes.Equal(rkgd[:4], rkgdFileMagic) {
		logging.Error(moduleName, "Invalid RKGD magic:", aurora.Cyan(string(rkgd[:4])))
		return false
	}

	expectedChecksum := binary.BigEndian.Uint32(rkgd[rkgdFileLength-4:])
	checksum := crc32.ChecksumIEEE(rkgd[:rkgdFileLength-4])

	if checksum != expectedChecksum {
		logging.Error(moduleName, "Invalid RKGD checksum:", aurora.Cyan(checksum), "expected:", aurora.Cyan(expectedChecksum))
		return false
	}

	lapCount := rkgd.GetLapCount()
	// This will need to be changed if/when we add support for tournament ghosts
	// Always make sure this value <= 5 due to a stack buffer overflow in the game's code
	if lapCount != 3 {
		logging.Error(moduleName, "Invalid RKGD lap count:", aurora.Cyan(lapCount))
		return false
	}

	lapScore := 0
	for lap := 0; lap <= lapCount; lap++ {
		if rkgd.GetTime(lap) == 0 {
			logging.Error(moduleName, "Zero RKGD time for lap:", aurora.Cyan(lap))
			return false
		}

		if rkgd.GetMinutes(lap) > 5 || rkgd.GetSeconds(lap) > 59 || rkgd.GetMilliseconds(lap) > 999 {
			logging.Error(moduleName, "Invalid RKGD time for lap m =", aurora.Cyan(rkgd.GetMinutes(lap)), "s =", aurora.Cyan(rkgd.GetSeconds(lap)), "ms =", aurora.Cyan(rkgd.GetMilliseconds(lap)))
			return false
		}

		if lap > 0 {
			lapScore += rkgd.GetTime(lap)
		}
	}

	totalScore := rkgd.GetTime(0)

	if expectedScore != -1 && totalScore != expectedScore {
		logging.Error(moduleName, "RKGD total score mismatch:", aurora.Cyan(rkgd.GetTime(0)), "expected:", aurora.Cyan(expectedScore))
		return false
	}

	// We'll do a grace of ~1 millisecond for the lap score compared to the total score,
	// in case there is a rounding error in the game's code
	if lapScore+1 < totalScore || lapScore-1 > totalScore {
		logging.Error(moduleName, "RKGD lap score mismatch:", aurora.Cyan(lapScore), "expected:", aurora.Cyan(expectedScore))
		return false
	}

	if expectedCourse != -1 && rkgd.GetCourse() != expectedCourse {
		logging.Error(moduleName, "RKGD course mismatch:", aurora.Cyan(rkgd.GetCourse()), "expected:", aurora.Cyan(expectedCourse))
		return false
	}

	if !rkgd.GetCourse().IsValid() {
		logging.Error(moduleName, "Invalid RKGD course:", aurora.Cyan(rkgd.GetCourse()))
		return false
	}

	if !rkgd.GetCharacter().IsValid() {
		logging.Error(moduleName, "Invalid RKGD character:", aurora.Cyan(rkgd.GetCharacter()))
		return false
	}

	if !rkgd.GetVehicle().IsValid() {
		logging.Error(moduleName, "Invalid RKGD vehicle:", aurora.Cyan(rkgd.GetVehicle()))
		return false
	}

	if rkgd.GetCharacter().GetWeightClass() != rkgd.GetVehicle().GetWeightClass() {
		logging.Error(moduleName, "RKGD character/vehicle weight class mismatch: c =", aurora.Cyan(rkgd.GetCharacter()), "v =", aurora.Cyan(rkgd.GetVehicle()))
	}

	if !rkgd.GetController().IsValid() {
		logging.Error(moduleName, "Invalid RKGD controller:", aurora.Cyan(rkgd.GetController()))
		return false
	}

	if rkgd.GetMiiData().RFLCalculateCRC() != 0x0000 {
		logging.Error(moduleName, "Invalid RKGD Mii data CRC")
		return false
	}

	// RKG uploaded to the server must be SZS (Yaz1) compressed
	if !rkgd.IsCompressed() {
		logging.Error(moduleName, "RKGD is not compressed")
		return false
	}

	szsData := rkgd.GetCompressedData()

	if string(szsData[:0x4]) != "Yaz1" {
		logging.Error(moduleName, "Invalid Yaz1 magic:", aurora.Cyan(string(szsData[:4])))
		return false
	}

	decompSize := binary.BigEndian.Uint32(szsData[0x4:0x8])
	if uint32(rkgd.GetInputDataLength()) != decompSize {
		logging.Error(moduleName, "Invalid RKGD input data length:", aurora.Cyan(rkgd.GetInputDataLength()), "actual:", aurora.Cyan(len(szsData)))
	}

	if rkgd.GetCompressedSize() != uint32(len(szsData)) {
		logging.Error(moduleName, "Invalid RKGD compressed size:", aurora.Cyan(rkgd.GetCompressedSize()), "actual:", aurora.Cyan(len(szsData)))
	}

	if !bytes.Equal(szsData[0x8:0x10], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) {
		logging.Error(moduleName, "Invalid SZS header padding")
	}

	valid, consumed := VerifyYaz1Data(moduleName, szsData[0x10:], int(decompSize), 0)

	if !valid {
		return false
	}

	if consumed+3 < len(szsData)-0x10 {
		logging.Error(moduleName, "Too much padding at end of RKGD")
		return false
	}

	return true
}

// Verify that SZS compressed data fits the standard. A buffer overflow bug is the basis for a critical RCE vulnerability in the game (szsHaxx).
func VerifyYaz1Data(moduleName string, szsData []byte, expectedDecompSize int, decoded int) (bool, int) {
	i := 0
	for decoded < expectedDecompSize {
		if i >= len(szsData) {
			logging.Error(moduleName, "Yaz1: Unexpected end of data")
			return false, i
		}

		flags := szsData[i]
		i++

		// This happens a lot, so might as well check for the sake of performance
		if flags == 0xFF {
			decoded += 8
			i += 8
			continue
		}

		for j := 0; j < 8; j++ {
			if flags&0x80 == 0 {
				if i+1 >= len(szsData) {
					logging.Error(moduleName, "Yaz1: Unexpected end of data")
					return false, i
				}

				copyLen := (szsData[i] >> 4) + 2

				copySrc := (int(szsData[i])&0x0F)<<8 | int(szsData[i+1])
				copySrc = decoded - copySrc - 1

				i += 2

				if copySrc < 0 {
					logging.Error(moduleName, "Yaz1: Copy source is out of bounds")
					return false, i
				}

				if copyLen == 2 {
					if i >= len(szsData) {
						logging.Error(moduleName, "Yaz1: Unexpected end of data")
						return false, i
					}

					copyLen = szsData[i] + 0x12
					i += 1
				}

				decoded += int(copyLen)
			} else {
				decoded++
				i++
			}

			if decoded >= expectedDecompSize {
				break
			}

			flags <<= 1
		}

	}

	if decoded > expectedDecompSize {
		logging.Error(moduleName, "Yaz1: Overran expected decompressed size")
		return false, i
	}

	if i > len(szsData) {
		logging.Error(moduleName, "Yaz1: Unexpected end of data")
		return false, i
	}

	return true, i
}
