package gamestats

// Expectations generated from the standalone reference implementation; save data is synthetic.

import (
	"encoding/base64"
	"strings"
	"testing"
)

const gtacwarsdsBlobLength = 3474

func gtacwarsdsTestBlob(storySeconds uint32, lions byte, cameras int) []byte {
	blob := make([]byte, gtacwarsdsBlobLength)
	gtacwarsdsPutU32(blob, gtacwarsdsIdentityOffset, 0x502B76C1)
	gtacwarsdsPutU32(blob, gtacwarsdsStoryCompleteOffset, storySeconds)
	blob[gtacwarsdsLionsOffset-gtacwarsdsSaveBlobBase] = lions

	for i := 0; i < cameras; i++ {
		blob[gtacwarsdsCameraBitmapOffset-gtacwarsdsSaveBlobBase+(i>>3)] |= 1 << (i & 7)
	}

	return blob
}

func gtacwarsdsPutU32(blob []byte, saveOffset int, value uint32) {
	offset := saveOffset - gtacwarsdsSaveBlobBase
	blob[offset] = byte(value)
	blob[offset+1] = byte(value >> 8)
	blob[offset+2] = byte(value >> 16)
	blob[offset+3] = byte(value >> 24)
}

func gtacwarsdsTestStored(blob []byte) map[string]string {
	return map[string]string{
		"SAVE": base64.StdEncoding.EncodeToString(blob),
		"FKEY": "1",
	}
}

var gtacwarsdsTestKeys = []string{"SAVEVER", ".DLKEY00", ".DLKEY01", ".DLKEY02", ".DLKEY05", ".DLKEY08", ".DLKEY09", ".DLKEY10", "FKEY"}

var gtacwarsdsGateCases = []struct {
	name         string
	storySeconds uint32
	lions        byte
	cameras      int
	wantOK       bool
}{
	{"fresh", 0, 0x00, 0, false},
	{"story only", 50448, 0x00, 100, false},
	{"one Lion", 50448, 0x01, 100, false},
	{"both Lions", 50448, 0x02, 100, true},
	{"Lions only", 0, 0x02, 0, false},
	{"100% reference", 50448, 0x02, 100, true},
}

func TestGtacwarsdsProgressGate(t *testing.T) {
	for _, tc := range gtacwarsdsGateCases {
		t.Run(tc.name, func(t *testing.T) {
			blob := gtacwarsdsTestBlob(tc.storySeconds, tc.lions, tc.cameras)

			got, reason := gtacwarsdsProgressGateOK(blob, gtacwarsdsDefaultGate)
			if got != tc.wantOK {
				t.Fatalf("gate = %v (%s), want %v", got, reason, tc.wantOK)
			}
		})
	}
}

var gtacwarsdsSeanGateCases = []struct {
	name    string
	cameras int
	wantOK  bool
}{
	{"no cameras", 0, false},
	{"one short", 92, false},
	{"all cameras", 100, true},
}

func TestGtacwarsdsSeanGate(t *testing.T) {
	for _, tc := range gtacwarsdsSeanGateCases {
		t.Run(tc.name, func(t *testing.T) {
			blob := gtacwarsdsTestBlob(50448, 0x02, tc.cameras)

			got, reason := gtacwarsdsProgressGateOK(blob, gtacwarsdsDefaultSeanGate)
			if got != tc.wantOK {
				t.Fatalf("sean gate = %v (%s), want %v", got, reason, tc.wantOK)
			}
		})
	}
}

var gtacwarsdsDecisionCases = []struct {
	name    string
	story   uint32
	lions   byte
	cameras int
	want    map[string]string
}{
	{
		name:    "fresh",
		story:   0,
		lions:   0x00,
		cameras: 0,
		want: map[string]string{
			"SAVEVER":  "1345025729",
			".DLKEY00": "",
			".DLKEY01": "",
			".DLKEY02": "1",
			".DLKEY05": "",
			".DLKEY08": "1",
			".DLKEY09": "1",
			".DLKEY10": "1",
			"FKEY":     "1",
		},
	},
	{
		name:    "story only",
		story:   50448,
		lions:   0x00,
		cameras: 100,
		want: map[string]string{
			"SAVEVER":  "1345025729",
			".DLKEY00": "",
			".DLKEY01": "1",
			".DLKEY02": "1",
			".DLKEY05": "",
			".DLKEY08": "1",
			".DLKEY09": "1",
			".DLKEY10": "1",
			"FKEY":     "1",
		},
	},
	{
		name:    "one Lion",
		story:   50448,
		lions:   0x01,
		cameras: 100,
		want: map[string]string{
			"SAVEVER":  "1345025729",
			".DLKEY00": "",
			".DLKEY01": "1",
			".DLKEY02": "1",
			".DLKEY05": "",
			".DLKEY08": "1",
			".DLKEY09": "1",
			".DLKEY10": "1",
			"FKEY":     "1",
		},
	},
	{
		name:    "Lions only",
		story:   0,
		lions:   0x02,
		cameras: 0,
		want: map[string]string{
			"SAVEVER":  "1345025729",
			".DLKEY00": "",
			".DLKEY01": "",
			".DLKEY02": "1",
			".DLKEY05": "",
			".DLKEY08": "1",
			".DLKEY09": "1",
			".DLKEY10": "1",
			"FKEY":     "1",
		},
	},
	{
		name:    "cameras short",
		story:   50448,
		lions:   0x02,
		cameras: 92,
		want: map[string]string{
			"SAVEVER":  "1345025729",
			".DLKEY00": "1",
			".DLKEY01": "",
			".DLKEY02": "1",
			".DLKEY05": "",
			".DLKEY08": "1",
			".DLKEY09": "1",
			".DLKEY10": "1",
			"FKEY":     "1",
		},
	},
	{
		name:    "100% reference",
		story:   50448,
		lions:   0x02,
		cameras: 100,
		want: map[string]string{
			"SAVEVER":  "1345025729",
			".DLKEY00": "1",
			".DLKEY01": "1",
			".DLKEY02": "1",
			".DLKEY05": "",
			".DLKEY08": "1",
			".DLKEY09": "1",
			".DLKEY10": "1",
			"FKEY":     "1",
		},
	},
}

func TestGtacwarsdsDecide(t *testing.T) {
	for _, tc := range gtacwarsdsDecisionCases {
		t.Run(tc.name, func(t *testing.T) {
			blob := gtacwarsdsTestBlob(tc.story, tc.lions, tc.cameras)
			stored := gtacwarsdsTestStored(blob)

			got := gtacwarsdsDecide(gtacwarsdsTestKeys, stored, gtacwarsdsDefaultMask, gtacwarsdsDefaultGate, gtacwarsdsDefaultSeanGate)

			for key, want := range tc.want {
				if got[key] != want {
					t.Errorf("%s = %q, want %q", key, got[key], want)
				}
			}

			if len(got) != len(tc.want) {
				t.Errorf("got %d keys, want %d", len(got), len(tc.want))
			}
		})
	}
}

// A fixed literal closes the client's own reward gate.
func TestGtacwarsdsSaveverEchoesUpload(t *testing.T) {
	blob := gtacwarsdsTestBlob(50448, 0x02, 100)

	if got, want := gtacwarsdsSavever(blob), "1345025729"; got != want {
		t.Fatalf(".SAVEVER = %q, want %q", got, want)
	}

	other := make([]byte, gtacwarsdsBlobLength)
	gtacwarsdsPutU32(other, gtacwarsdsIdentityOffset, 0xDEADBEEF)

	if got, want := gtacwarsdsSavever(other), "3735928559"; got != want {
		t.Fatalf(".SAVEVER = %q, want %q", got, want)
	}

	if got := gtacwarsdsSavever(make([]byte, 8)); got != "" {
		t.Fatalf(".SAVEVER = %q, want empty", got)
	}
}

// Not used to make the decision, but a reward prediction is checked against it.
func TestGtacwarsdsRewardBitMap(t *testing.T) {
	want := map[int]int{
		0:  4,  // Xin Shan missions
		1:  2,  // Sean (81st dealer)
		2:  0,  // Bulletproof Patriot
		3:  6,  // unidentified
		5:  10, // $10,000
		6:  12, // unidentified
		7:  14, // unidentified
		8:  16, // Bulletproof Infernus
		9:  18, // Bulletproof Hellenbach
		10: 20, // Bulletproof Cavalcade FXT
	}

	if len(gtacwarsdsDLKeyRewardBit) != len(want) {
		t.Fatalf("map has %d entries, want %d", len(gtacwarsdsDLKeyRewardBit), len(want))
	}

	for index, bit := range want {
		if gtacwarsdsDLKeyRewardBit[index] != bit {
			t.Errorf(".DLKEY%02d -> bit %d, want %d", index, gtacwarsdsDLKeyRewardBit[index], bit)
		}
	}
}

// The reply must carry only the requested keys; .DLKEY11 awards money.
func TestGtacwarsdsPublicDataShape(t *testing.T) {
	blob := gtacwarsdsTestBlob(50448, 0x02, 100)
	stored := gtacwarsdsTestStored(blob)

	var wire strings.Builder
	for key, value := range stored {
		wire.WriteString(`\` + key + `\` + value)
	}

	got := gtacwarsdsPublicData("GSTATS:test", "\x01.DLKEY00\x01FKEY", wire.String())

	want := `\.DLKEY00\1\FKEY\1`
	if got != want {
		t.Fatalf("public data = %q, want %q", got, want)
	}

	refused := gtacwarsdsPublicData("GSTATS:test", "\x01.DLKEY00", gtacwarsdsWire(gtacwarsdsTestStored(gtacwarsdsTestBlob(0, 0x00, 0))))
	if refused != `\.DLKEY00\` {
		t.Fatalf("refused public data = %q, want %q", refused, `\.DLKEY00\`)
	}

	if got := gtacwarsdsPublicData("GSTATS:test", "", wire.String()); got != wire.String() {
		t.Fatalf("empty key list should echo the stored profile, got %q", got)
	}
}

func gtacwarsdsWire(stored map[string]string) string {
	var wire strings.Builder
	for key, value := range stored {
		wire.WriteString(`\` + key + `\` + value)
	}
	return wire.String()
}

func TestGtacwarsdsShortBlobRefused(t *testing.T) {
	short := make([]byte, 100)

	ok, reason := gtacwarsdsProgressGateOK(short, gtacwarsdsDefaultGate)
	if ok {
		t.Fatalf("gate allowed a 100-byte blob (%s)", reason)
	}

	if got := gtacwarsdsDecide([]string{".DLKEY00"}, map[string]string{"SAVE": base64.StdEncoding.EncodeToString(short)}, gtacwarsdsDefaultMask, gtacwarsdsDefaultGate, gtacwarsdsDefaultSeanGate); got[".DLKEY00"] != "" {
		t.Fatalf("served .DLKEY00 for an unusable blob: %q", got[".DLKEY00"])
	}

	if ok, _ := gtacwarsdsProgressGateOK(nil, gtacwarsdsDefaultGate); ok {
		t.Fatal("gate allowed a session with no uploaded save")
	}
}

// An unparsable rule allows the grant by design.
func TestGtacwarsdsGateSpec(t *testing.T) {
	complete := gtacwarsdsTestBlob(50448, 0x02, 100)
	fresh := gtacwarsdsTestBlob(0, 0x00, 0)

	cases := []struct {
		name string
		spec string
		blob []byte
		want bool
	}{
		{"off serves unconditionally", "off", fresh, true},
		{"empty spec serves unconditionally", "", fresh, true},
		{"story rule satisfied", "0x3C4:!=0", complete, true},
		{"story rule violated", "0x3C4:!=0", fresh, false},
		{"equality rule satisfied", "0x3C4:==50448", complete, true},
		{"equality rule violated", "0x3C4:==50448", fresh, false},
		{"masked rule satisfied", "0x424:0x02:0x02", complete, true},
		{"masked rule violated", "0x424:0x01:0x01", complete, false},
		{"bit run satisfied", "0x684:bits:100", complete, true},
		{"bit run violated", "0x684:bits:99", complete, false},
		{"both rules must hold", gtacwarsdsDefaultGate, fresh, false},
		{"both rules hold", gtacwarsdsDefaultGate, complete, true},
		{"unparsable offset allows", "zzz:!=0", fresh, true},
		{"unparsable value allows", "0x3C4:!=zzz", fresh, true},
		{"unparsable bit count allows", "0x684:bits:zzz", fresh, true},
		{"unknown form allows", "0x3C4", fresh, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := gtacwarsdsProgressGateOK(tc.blob, tc.spec)
			if got != tc.want {
				t.Fatalf("gate = %v (%s), want %v", got, reason, tc.want)
			}

			if reason == "" {
				t.Error("every decision needs a reason for the log")
			}
		})
	}
}
