package common

type MarioKartWiiLeaderboardRegionId int
type MarioKartWiiCourseId int
type MarioKartWiiControllerId int

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

const (
	WiiWheel             = iota // 0x00
	WiiRemoteAndNunchuck        // 0x01
	Classic                     // 0x02
	GameCube                    // 0x03
)

func (regionId MarioKartWiiLeaderboardRegionId) IsValid() bool {
	return regionId >= Worldwide && regionId <= China
}

func (courseId MarioKartWiiCourseId) IsValid() bool {
	return courseId >= MarioCircuit && courseId <= GBAShyGuyBeach
}

func (controllerId MarioKartWiiControllerId) IsValid() bool {
	return controllerId >= WiiWheel && controllerId <= GameCube
}
