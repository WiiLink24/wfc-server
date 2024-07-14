package common

type MarioKartWiiRegionID int
type MarioKartWiiCourseID int

const MarioKartWiiGameSpyGameID int = 1687

const (
	Worldwide    = iota // 0
	Japan        = iota // 1
	UnitedStates = iota // 2
	Europe       = iota // 3
	Australia    = iota // 4
	Taiwan       = iota // 5
	Korea        = iota // 6
	China        = iota // 7
)

const (
	MarioCircuit        = iota // 0x00
	MooMooMeadows       = iota // 0x01
	MushroomGorge       = iota // 0x02
	GrumbleVolcano      = iota // 0x03
	ToadsFactory        = iota // 0x04
	CoconutMall         = iota // 0x05
	DKSummit            = iota // 0x06
	WarioGoldMine       = iota // 0x07
	LuigiCircuit        = iota // 0x08
	DaisyCircuit        = iota // 0x09
	MoonviewHighway     = iota // 0x0A
	MapleTreeway        = iota // 0x0B
	BowsersCastle       = iota // 0x0C
	RainbowRoad         = iota // 0x0D
	DryDryRuins         = iota // 0x0E
	KoopaCape           = iota // 0x0F
	GCNPeachBeach       = iota // 0x10
	GCNMarioCircuit     = iota // 0x11
	GCNWaluigiStadium   = iota // 0x12
	GCNDKMountain       = iota // 0x13
	DSYoshiFalls        = iota // 0x14
	DSDesertHills       = iota // 0x15
	DSPeachGardens      = iota // 0x16
	DSDelfinoSquare     = iota // 0x17
	SNESMarioCircuit3   = iota // 0x18
	SNESGhostValley2    = iota // 0x19
	N64MarioRaceway     = iota // 0x1A
	N64SherbetLand      = iota // 0x1B
	N64BowsersCastle    = iota // 0x1C
	N64DKsJungleParkway = iota // 0x1D
	GBABowserCastle3    = iota // 0x1E
	GBAShyGuyBeach      = iota // 0x1F
)

func (regionId MarioKartWiiRegionID) IsValid() bool {
	return regionId >= Worldwide && regionId <= China
}

func (courseId MarioKartWiiCourseID) IsValid() bool {
	return courseId >= MarioCircuit && courseId <= GBAShyGuyBeach
}

func (regionId MarioKartWiiRegionID) ToString() string {
	return [...]string{
		"Worldwide",
		"Japan",
		"United States",
		"Europe",
		"Australia",
		"Taiwan",
		"Korea",
		"China",
	}[regionId]
}

func (courseId MarioKartWiiCourseID) ToString() string {
	return [...]string{
		"Mario Circuit",
		"Moo Moo Meadows",
		"Mushroom Gorge",
		"Grumble Volcano",
		"Toad's Factory",
		"Coconut Mall",
		"DK Summit",
		"Wario's Gold Mine",
		"Luigi Circuit",
		"Daisy Circuit",
		"Moonview Highway",
		"Maple Treeway",
		"Bowser's Castle",
		"Rainbow Road",
		"Dry Dry Ruins",
		"Koopa Cape",
		"GCN Peach Beach",
		"GCN Mario Circuit",
		"GCN Waluigi Stadium",
		"GCN DK Mountain",
		"DS Yoshi Falls",
		"DS Desert Hills",
		"DS Peach Gardens",
		"DS Delfino Square",
		"SNES Mario Circuit 3",
		"SNES Ghost Valley 2",
		"N64 Mario Raceway",
		"N64 Sherbet Land",
		"N64 Bowser's Castle",
		"N64 DK's Jungle Parkway",
		"GBA Bowser Castle 3",
		"GBA Shy Guy Beach",
	}[courseId]
}
