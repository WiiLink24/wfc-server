package gamestats

import (
	"encoding/base64"
	"encoding/binary"
	"strconv"
	"strings"

	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

// GTA: Chinatown Wars unlocks the Xin Shan missions through the Rockstar Social Club, which is dead.
// The client never uploads the .DLKEY00-15 values it asks for, so they are computed from its save.

const (
	gtacwarsdsSaveBlobBase        = 0x1EC // uploaded save starts here
	gtacwarsdsStoryCompleteOffset = 0x3C4 // "time to complete story", 0 if unfinished
	gtacwarsdsLionsOffset         = 0x424 // Lions of Fo collected, 2 = both
	gtacwarsdsCameraBitmapOffset  = 0x684 // 100 bits, one per security camera
	gtacwarsdsCameraCount         = 100
	gtacwarsdsIdentityOffset      = 0x230 // served as .SAVEVER

	// Story complete and both Lions of Fo. Sean additionally needs every camera.
	gtacwarsdsDefaultGate     = "0x3C4:!=0,0x424:0x02:0x02"
	gtacwarsdsDefaultSeanGate = "0x684:bits:100"

	gtacwarsdsDefaultMask  = 0x0707 // the four bulletproof vehicles, Sean and Xin
	gtacwarsdsGrantedValue = "1"
)

// Reward bit for each .DLKEY index in the reward word at save 0x6B0. .DLKEY04 writes nothing and
// .DLKEY11 is a money clamp, not a reward.
var gtacwarsdsDLKeyRewardBit = map[int]int{
	0:  4,  // Xin Shan missions
	1:  2,  // Sean, the 81st dealer
	2:  0,  // Bulletproof Patriot
	3:  6,  // unidentified (also writes save 0x0DF3)
	5:  10, // $10,000 (Ammu-Nation reward)
	6:  12, // unidentified
	7:  14, // unidentified
	8:  16, // Bulletproof Infernus
	9:  18, // Bulletproof Hellenbach
	10: 20, // Bulletproof Cavalcade FXT
}

func gtacwarsdsParsePairs(message string) map[string]string {
	fields := strings.Split(message, `\`)

	pairs := make(map[string]string, len(fields)/2)
	for i := 1; i+1 < len(fields); i += 2 {
		pairs[fields[i]] = fields[i+1]
	}

	return pairs
}

func gtacwarsdsDecodeSaveBlob(blobB64 string) []byte {
	if blobB64 == "" {
		return nil
	}

	if remainder := len(blobB64) % 4; remainder != 0 {
		blobB64 += strings.Repeat("=", 4-remainder)
	}

	raw, err := base64.StdEncoding.DecodeString(blobB64)
	if err != nil || len(raw) == 0 {
		return nil
	}

	return raw
}

func gtacwarsdsCountBits(blob []byte, count int) uint64 {
	var set uint64

	for i := 0; i < count; i++ {
		if blob[i>>3]>>(i&7)&1 == 1 {
			set++
		}
	}

	return set
}

// Rules are comma-separated, all must hold, and an unparsable one allows the grant rather than
// denying a player over a typo. Offsets are save offsets; 0x424:0x02:0x02 is a masked byte and
// 0x684:bits:100 counts a 100-bit run.
func gtacwarsdsProgressGateOK(blob []byte, spec string) (bool, string) {
	if strings.TrimSpace(strings.ToLower(spec)) == "" || strings.EqualFold(strings.TrimSpace(spec), "off") {
		return true, "gate off"
	}

	if blob == nil {
		return false, "no uploaded save to check"
	}

	for _, rule := range strings.Split(spec, ",") {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		parts := strings.Split(rule, ":")

		offset, err := strconv.ParseInt(parts[0], 0, 64)
		if err != nil {
			return true, "unparsable rule " + rule + " - allowing"
		}

		blobOffset := int(offset) - gtacwarsdsSaveBlobBase

		switch {
		case len(parts) == 2 && (strings.HasPrefix(parts[1], "!=") || strings.HasPrefix(parts[1], "==")):
			if blobOffset < 0 || blobOffset+4 > len(blob) {
				return false, "save offset 0x" + strconv.FormatInt(offset, 16) + " outside the uploaded blob"
			}

			got := binary.LittleEndian.Uint32(blob[blobOffset:])

			want, err := strconv.ParseUint(parts[1][2:], 0, 32)
			if err != nil {
				return true, "unparsable rule " + rule + " - allowing"
			}

			if strings.HasPrefix(parts[1], "!=") && got == uint32(want) {
				return false, "save[0x" + strconv.FormatInt(offset, 16) + "] = " + strconv.FormatUint(uint64(got), 10) +
					", must differ from " + strconv.FormatUint(want, 10)
			}

			if strings.HasPrefix(parts[1], "==") && got != uint32(want) {
				return false, "save[0x" + strconv.FormatInt(offset, 16) + "] = " + strconv.FormatUint(uint64(got), 10) +
					", need " + strconv.FormatUint(want, 10)
			}

		case len(parts) == 3 && strings.EqualFold(parts[1], "bits"):
			want, err := strconv.ParseUint(parts[2], 0, 32)
			if err != nil {
				return true, "unparsable rule " + rule + " - allowing"
			}

			if blobOffset < 0 || blobOffset+(gtacwarsdsCameraCount+7)/8 > len(blob) {
				return false, "save offset 0x" + strconv.FormatInt(offset, 16) + " outside the uploaded blob"
			}

			if got := gtacwarsdsCountBits(blob[blobOffset:], gtacwarsdsCameraCount); got != want {
				return false, "save[0x" + strconv.FormatInt(offset, 16) + "] has " + strconv.FormatUint(got, 10) +
					" of " + strconv.Itoa(gtacwarsdsCameraCount) + " bits set, need " + strconv.FormatUint(want, 10)
			}

		case len(parts) == 3:
			mask, err := strconv.ParseUint(parts[1], 0, 8)
			if err != nil {
				return true, "unparsable rule " + rule + " - allowing"
			}

			value, err := strconv.ParseUint(parts[2], 0, 8)
			if err != nil {
				return true, "unparsable rule " + rule + " - allowing"
			}

			if blobOffset < 0 || blobOffset >= len(blob) {
				return false, "save offset 0x" + strconv.FormatInt(offset, 16) + " outside the uploaded blob"
			}

			if got := blob[blobOffset] & uint8(mask); got != uint8(value) {
				return false, "save[0x" + strconv.FormatInt(offset, 16) + "]&0x" + strconv.FormatUint(mask, 16) +
					" = 0x" + strconv.FormatUint(uint64(got), 16) + ", need 0x" + strconv.FormatUint(value, 16)
			}

		default:
			return true, "unparsable rule " + rule + " - allowing"
		}
	}

	return true, "passed"
}

// The reward writer only commits a reward bit when [0x021f10cc + 0x70] matches [profile + 0x2d0],
// which .SAVEVER holds, so this has to echo the client's own value and not a literal.
func gtacwarsdsSavever(blob []byte) string {
	offset := gtacwarsdsIdentityOffset - gtacwarsdsSaveBlobBase
	if blob == nil || len(blob) < offset+4 {
		return ""
	}

	return strconv.FormatUint(uint64(binary.LittleEndian.Uint32(blob[offset:])), 10)
}

func gtacwarsdsDecide(keys []string, stored map[string]string, mask int, gate string, seanGate string) map[string]string {
	blob := gtacwarsdsDecodeSaveBlob(stored["SAVE"])
	gateOK, _ := gtacwarsdsProgressGateOK(blob, gate)
	seanGateOK, _ := gtacwarsdsProgressGateOK(blob, seanGate)

	decided := make(map[string]string, len(keys))

	for _, key := range keys {
		if key == "" {
			continue
		}

		value := stored[key]

		if key == "SAVEVER" {
			decided[key] = gtacwarsdsSavever(blob)
			continue
		}

		name := strings.TrimPrefix(key, ".")

		if strings.HasPrefix(name, "DLKEY") {
			index, err := strconv.Atoi(name[len("DLKEY"):])
			if err != nil {
				decided[key] = value
				continue
			}

			if (mask>>index)&1 == 1 {
				switch {
				case index == 0 && !gateOK:
					value = ""
				case index == 1 && !seanGateOK:
					value = ""
				default:
					value = gtacwarsdsGrantedValue
				}
			}

			decided[key] = value
			continue
		}

		decided[key] = value
	}

	return decided
}

// Only the keys the client asked for are carried; answering with others can award money.
func gtacwarsdsPublicData(moduleName string, keysField string, storedData string) string {
	stored := gtacwarsdsParsePairs(storedData)

	var keys []string
	for _, key := range strings.Split(keysField, "\x01") {
		if key != "" {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return storedData
	}

	blob := gtacwarsdsDecodeSaveBlob(stored["SAVE"])

	for _, key := range keys {
		switch key {
		case ".DLKEY00":
			gateOK, gateReason := gtacwarsdsProgressGateOK(blob, gtacwarsdsDefaultGate)

			logging.Info(moduleName, "gtacwarsds/GetPD: Xin entitlement served:", aurora.Cyan(gateOK), "reason:", aurora.Cyan(gateReason))
		case ".DLKEY01":
			seanOK, seanReason := gtacwarsdsProgressGateOK(blob, gtacwarsdsDefaultSeanGate)

			logging.Info(moduleName, "gtacwarsds/GetPD: Sean entitlement served:", aurora.Cyan(seanOK), "reason:", aurora.Cyan(seanReason))
		}
	}

	decided := gtacwarsdsDecide(keys, stored, gtacwarsdsDefaultMask, gtacwarsdsDefaultGate, gtacwarsdsDefaultSeanGate)

	var data strings.Builder
	for _, key := range keys {
		data.WriteString(`\`)
		data.WriteString(key)
		data.WriteString(`\`)
		data.WriteString(decided[key])
	}

	return data.String()
}
