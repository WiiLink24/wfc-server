package nas

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

var (
	dlcDir = "./dlc"
)

func handleAuthRequest(moduleName string, w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logging.Error(moduleName, "Failed to parse form")
		replyHTTPError(w, 400, "400 Bad Request")
		return
	}

	// Need to know this here to determine UTF-16 endianness (LE for DS, BE for Wii)
	// unitcd 0 = DS, 1 = Wii
	unitcd := "1"
	if unitcdValues, ok := r.PostForm["unitcd"]; ok {
		unitcdDecoded, err := common.Base64DwcEncoding.DecodeString(unitcdValues[0])
		if err != nil {
			logging.Error(moduleName, "Invalid unitcd string in form")
			replyHTTPError(w, 400, "400 Bad Request")
			return
		}
		unitcd = string(unitcdDecoded)
	}

	fields := map[string]string{}
	for key, values := range r.PostForm {
		if len(values) != 1 {
			logging.Warn(moduleName, "Ignoring none or multiple POST form values:", aurora.Cyan(key).String()+":", aurora.Cyan(values))
			continue
		}

		var value string
		if !strings.HasPrefix(key, "_") {
			parsed, err := common.Base64DwcEncoding.DecodeString(values[0])
			if err != nil {
				logging.Error(moduleName, "Invalid POST form value:", aurora.Cyan(key).String()+":", aurora.Cyan(values[0]))
				replyHTTPError(w, 400, "400 Bad Request")
				return
			}

			if key == "ingamesn" || key == "devname" || key == "words" {
				// Special handling required for the UTF-16 string
				var utf16String []uint16
				if unitcd == "0" {
					for i := 0; i < len(parsed)/2; i++ {
						utf16String = append(utf16String, binary.LittleEndian.Uint16(parsed[i*2:i*2+2]))
					}
				} else {
					for i := 0; i < len(parsed)/2; i++ {
						utf16String = append(utf16String, binary.BigEndian.Uint16(parsed[i*2:i*2+2]))
					}
				}
				value = string(utf16.Decode(utf16String))
			} else {
				value = string(parsed)
			}
		} else {
			// Values unique to CTGP/the Wiimmfi payload, for compatibility reasons. Some of these are not base64 encoded.
			value = values[0]
		}

		logging.Info(moduleName, aurora.Cyan(key).String()+":", aurora.Cyan(value))
		fields[key] = value
	}

	reply := map[string]string{}
	var response []byte

	if r.URL.String() == "/ac" {
		action, ok := fields["action"]
		if !ok || action == "" {
			logging.Error(moduleName, "No action in form")
			replyHTTPError(w, 400, "400 Bad Request")
			return
		}

		switch strings.ToLower(action) {
		case "acctcreate":
			reply = acctcreate()

		case "login":
			isLocalhost := strings.HasPrefix(r.RemoteAddr, "127.0.0.1:") || strings.HasPrefix(r.RemoteAddr, "[::1]:")
			reply = login(moduleName, fields, isLocalhost)

		case "svcloc":
			reply = svcloc(fields)

		default:
			logging.Error(moduleName, "Unknown action:", aurora.Cyan(action))
			reply = map[string]string{
				"retry":    "0",
				"returncd": "109",
			}
		}
	} else if r.URL.String() == "/pr" {
		words, ok := fields["words"]
		if words == "" || !ok {
			logging.Error(moduleName, "No words in form")
			replyHTTPError(w, 400, "400 Bad Request")
			return
		}

		reply = handleProfanity(fields)
	} else if r.URL.String() == "/download" {
		action, ok := fields["action"]
		if !ok || action == "" {
			logging.Error(moduleName, "No action in form")
			replyHTTPError(w, 400, "400 Bad Request")
			return
		}

		rhgamecd, ok := fields["rhgamecd"]
		if !ok || !isValidRhgamecd(rhgamecd) {
			logging.Error(moduleName, "Missing or invalid rhgamecd")
			replyHTTPError(w, 400, "400 Bad Request")
			return
		}

		switch strings.ToLower(action) {
		case "count":
			response = []byte(dlsCount(fields))

		default:
			logging.Error(moduleName, "Unknown action:", aurora.Cyan(action))
			reply = map[string]string{
				"retry":    "0",
				"returncd": "109",
			}
		}

		w.Header().Set("X-DLS-Host", "http://127.0.0.1/")
	}

	if len(response) == 0 {
		param := url.Values{}
		for key, value := range reply {
			param.Set(key, common.Base64DwcEncoding.EncodeToString([]byte(value)))
		}
		response = []byte(param.Encode())
		response = []byte(strings.Replace(string(response), "%2A", "*", -1))
	}

	// DWC treats the response like a null terminated string
	response = append(response, 0x00)

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Write(response)
}

func acctcreate() map[string]string {
	return map[string]string{
		"retry":    "0",
		"datetime": getDateTime(),
		"returncd": "002",
		"userid":   strconv.FormatUint(database.GetUniqueUserID(), 10),
	}
}

func login(moduleName string, fields map[string]string, isLocalhost bool) map[string]string {
	param := map[string]string{
		"retry":    "0",
		"datetime": getDateTime(),
		"locator":  "gamespy.com",
	}

	gamecd, ok := fields["gamecd"]
	if !ok {
		logging.Error(moduleName, "No gamecd in form")
		param["returncd"] = "103"
		return param
	}

	strUserId, ok := fields["userid"]
	if !ok {
		logging.Error(moduleName, "No userid in form")
		param["returncd"] = "103"
		return param
	}

	userId, err := strconv.ParseUint(strUserId, 10, 64)
	if err != nil || userId >= 0x80000000000 {
		logging.Error(moduleName, "Invalid userid string in form")
		param["returncd"] = "103"
		return param
	}

	gsbrcd, ok := fields["gsbrcd"]
	if !ok {
		logging.Error(moduleName, "No gsbrcd in form")
		param["returncd"] = "103"
		return param
	}

	if (len(gsbrcd) < 4 && len(gsbrcd) != 0) || strings.ContainsRune(gsbrcd, 0) {
		logging.Error(moduleName, "Invalid gsbrcd string in form")
		param["returncd"] = "103"
		return param
	}

	// Some games like Fortune Street make login requests without a gsbr code, so we temporarily fake one
	if len(gsbrcd) == 0 {
		if len(gamecd) < 4 {
			logging.Error(moduleName, "Invalid gamecd string in form")
			param["returncd"] = "103"
			return param
		}

		gsbrcd = gamecd[:3] + "J"
	}

	lang, ok := fields["lang"]
	if !ok {
		lang = "ff"
	}

	langByte, err := hex.DecodeString(lang)
	if err != nil || len(langByte) != 1 {
		logging.Error(moduleName, "Invalid lang byte in form")
		param["returncd"] = "103"
		return param
	}

	unitcd, ok := fields["unitcd"]
	if !ok {
		logging.Error(moduleName, "No unitcd in form")
		param["returncd"] = "103"
		return param
	}

	unitcdInt, err := strconv.ParseUint(unitcd, 10, 64)
	if err != nil || unitcdInt > 1 {
		logging.Error(moduleName, "Invalid unitcd string in form")
		param["returncd"] = "103"
		return param
	}

	hasProfaneName := false
	ingamesn, ok := fields["ingamesn"]
	if ok {
		if hasProfaneName, _ = IsBadWord(ingamesn); hasProfaneName {
			logging.Info(moduleName, aurora.Cyan(strconv.FormatUint(userId, 10)), "has a profane name ("+aurora.Red(ingamesn).String()+")")
		}
	}

	var authToken, challenge string
	switch unitcdInt {
	// ds
	case 0:
		devname, ok := fields["devname"]
		if !ok {
			logging.Error(moduleName, "No devname in form")
			param["returncd"] = "103"
			return param
		}

		// Only later DS games send this
		ingamesn, ok := fields["ingamesn"]
		if ok {
			authToken, challenge = common.MarshalNASAuthToken(gamecd, userId, gsbrcd, 0, 0, langByte[0], ingamesn, 0, isLocalhost)
			logging.Notice(moduleName, "Login (DS)", aurora.Cyan(strconv.FormatUint(userId, 10)), aurora.Cyan(gsbrcd), "devname:", aurora.Cyan(devname), "ingamesn:", aurora.Cyan(ingamesn))
		} else {
			authToken, challenge = common.MarshalNASAuthToken(gamecd, userId, gsbrcd, 0, 0, langByte[0], "", 0, isLocalhost)
			logging.Notice(moduleName, "Login (DS)", aurora.Cyan(strconv.FormatUint(userId, 10)), aurora.Cyan(gsbrcd), "devname:", aurora.Cyan(devname))
		}

	// wii
	case 1:
		cfc, ok := fields["cfc"]
		if !ok {
			logging.Error(moduleName, "No cfc in form")
			param["returncd"] = "103"
			return param
		}

		cfcInt, err := strconv.ParseUint(cfc, 10, 64)
		if err != nil || cfcInt > 9999999999999999 {
			logging.Error(moduleName, "Invalid cfc string in form")
			param["returncd"] = "103"
			return param
		}

		region, ok := fields["region"]
		if !ok {
			region = "ff"
		}

		regionByte, err := hex.DecodeString(region)
		if err != nil || len(regionByte) != 1 {
			logging.Error(moduleName, "Invalid region byte in form")
			param["returncd"] = "103"
			return param
		}

		authToken, challenge = common.MarshalNASAuthToken(gamecd, userId, gsbrcd, cfcInt, regionByte[0], langByte[0], fields["ingamesn"], 1, isLocalhost)
		logging.Notice(moduleName, "Login (Wii)", aurora.Cyan(strconv.FormatUint(userId, 10)), aurora.Cyan(gsbrcd), "ingamesn:", aurora.Cyan(fields["ingamesn"]))
	}

	if hasProfaneName {
		param["returncd"] = "040"
	} else {
		param["returncd"] = "001"
	}
	param["challenge"] = challenge
	param["token"] = authToken

	return param
}

func svcloc(fields map[string]string) map[string]string {
	param := map[string]string{
		"retry":      "0",
		"datetime":   getDateTime(),
		"returncd":   "007",
		"statusdata": "Y",
	}

	authToken := "NDS/SVCLOC/TOKEN"

	switch fields["svc"] {
	default:
		param["servicetoken"] = authToken
		param["svchost"] = "n/a"

	case "9000":
		param["token"] = authToken
		param["svchost"] = "dls1.nintendowifi.net"

	case "9001":
		param["servicetoken"] = authToken
		param["svchost"] = "dls1.nintendowifi.net"
	}

	return param
}

func handleProfanity(fields map[string]string) map[string]string {
	var prwords string
	for _, word := range strings.Split(fields["words"], "\t") {
		if isBadWord, _ := IsBadWord(word); isBadWord {
			prwords += "1"
		} else {
			prwords += "0"
		}
	}

	var returncd string
	if strings.Contains(prwords, "1") {
		returncd = "040"
	} else {
		returncd = "000"
	}

	return map[string]string{
		"returncd": returncd,
		"prwords":  prwords,
	}
}

func dlsCount(fields map[string]string) string {
	dlcFolder := filepath.Join(dlcDir, fields["rhgamecd"])

	dir, ok := os.ReadDir(dlcFolder)
	if ok != nil {
		return "0"
	}

	return strconv.Itoa(len(dir))
}

func isValidRhgamecd(rhgamecd string) bool {
	if len(rhgamecd) != 4 {
		return false
	}

	return common.IsUppercaseAlphanumeric(rhgamecd)
}

func getDateTime() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
