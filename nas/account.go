package nas

import (
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

var accountActions = map[string]func(moduleName string, fields map[string][]byte) map[string]string{
	"acctcreate": acctcreate,
	"login":      login,
	"svcloc":     svcloc,
}

func handleAuthAccountEndpoint(w http.ResponseWriter, r *http.Request) {
	moduleName := getModuleName(r)

	fields, err := parseAuthRequest(r)
	if err != nil {
		replyHTTPError(w, 400, "400 Bad Request")
		return
	}

	action := string(fields["action"])
	if action == "" {
		logging.Error(moduleName, "No action in form")
		replyHTTPError(w, 400, "400 Bad Request")
		return
	}

	if actionFunc, exists := accountActions[strings.ToLower(action)]; exists {
		reply := actionFunc(moduleName, fields)
		writeAuthResponse(w, reply)
		return
	}

	logging.Error(moduleName, "Unknown action:", aurora.Cyan(action))
	replyHTTPError(w, 400, "400 Bad Request")
}

func acctcreate(moduleName string, fields map[string][]byte) map[string]string {
	return map[string]string{
		"retry":    "0",
		"datetime": getDateTime(),
		"returncd": "002",
		"userid":   strconv.FormatUint(database.GetUniqueUserID(), 10),
	}
}

func login(moduleName string, fields map[string][]byte) map[string]string {
	param := map[string]string{
		"retry":    "0",
		"datetime": getDateTime(),
		"locator":  "gamespy.com",
	}

	token := common.NASAuthToken{}

	gamecd, ok := fields["gamecd"]
	if !ok {
		logging.Error(moduleName, "No gamecd in form")
		param["returncd"] = "103"
		return param
	}
	copy(token.GameCode[:], gamecd)

	strUserId, ok := fields["userid"]
	if !ok {
		logging.Error(moduleName, "No userid in form")
		param["returncd"] = "103"
		return param
	}

	var err error
	token.UserID, err = strconv.ParseUint(string(strUserId), 10, 64)
	if err != nil || token.UserID >= 0x80000000000 {
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

	if (len(gsbrcd) < 4 && len(gsbrcd) != 0) || strings.ContainsRune(string(gsbrcd), 0) {
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

		gsbrcd = append(gamecd[:3], 'J')
	}

	copy(token.GsbrCode[:], gsbrcd)

	lang, ok := fields["lang"]
	if !ok {
		lang = []byte("ff")
	}

	langByte, err := hex.DecodeString(string(lang))
	if err != nil || len(langByte) != 1 {
		logging.Error(moduleName, "Invalid lang byte in form")
		param["returncd"] = "103"
		return param
	}
	token.Lang = langByte[0]

	unitcd, ok := fields["unitcd"]
	if !ok {
		logging.Error(moduleName, "No unitcd in form")
		param["returncd"] = "103"
		return param
	}

	isWii := len(unitcd) > 1 || unitcd[0] != '0'
	var endianness binary.ByteOrder
	switch isWii {
	case false:
		token.UnitCode = 0
		endianness = binary.LittleEndian
	case true:
		token.UnitCode = 1
		endianness = binary.BigEndian
	}

	hasProfaneName := false
	ingamesn, hasIngamesn := fields["ingamesn"]
	ingamesnStr := ""
	if hasIngamesn {
		ingamesnStr = common.UTF16Decode(ingamesn, endianness)
		if hasProfaneName, _ = IsBadWord(ingamesnStr); hasProfaneName {
			logging.Info(moduleName, "Provided in-game screen name has a profane word:", aurora.Red(ingamesnStr).String())
			// Continue with different return code
		}
	}

	switch isWii {
	case false:
		devname, ok := fields["devname"]
		if !ok {
			logging.Error(moduleName, "No devname in form")
			param["returncd"] = "103"
			return param
		}

		// Only later DS games send ingamesn
		if !hasIngamesn {
			ingamesn = devname
		}
		logging.Notice(moduleName, "Login (DS)", aurora.Cyan(token.UserID), aurora.Cyan(string(gsbrcd)), "devname:", aurora.Cyan(common.UTF16Decode(devname, endianness)), "name:", aurora.Cyan(ingamesnStr))

	case true:
		cfc, ok := fields["cfc"]
		if !ok {
			logging.Error(moduleName, "No cfc in form")
			param["returncd"] = "103"
			return param
		}

		token.ConsoleFriendCode, err = strconv.ParseUint(string(cfc), 10, 64)
		if err != nil || token.ConsoleFriendCode > 9999999999999999 {
			logging.Error(moduleName, "Invalid cfc string in form")
			param["returncd"] = "103"
			return param
		}

		region, ok := fields["region"]
		if !ok {
			region = []byte("ff")
		}

		regionByte, err := hex.DecodeString(string(region))
		if err != nil || len(regionByte) != 1 {
			logging.Error(moduleName, "Invalid region byte in form")
			param["returncd"] = "103"
			return param
		}
		token.Region = regionByte[0]

		logging.Notice(moduleName, "Login (Wii)", aurora.Cyan(token.UserID), aurora.Cyan(string(gsbrcd)), "name:", aurora.Cyan(ingamesnStr))
	}

	challenge := common.RandomString(8)
	copy(token.Challenge[:], []byte(challenge))
	copy(token.InGameScreenName[:], ingamesn)

	if hasProfaneName {
		param["returncd"] = "040"
	} else {
		param["returncd"] = "001"
	}

	param["challenge"] = challenge
	param["token"] = token.Marshal()

	return param
}

func svcloc(moduleName string, fields map[string][]byte) map[string]string {
	param := map[string]string{
		"retry":      "0",
		"datetime":   getDateTime(),
		"returncd":   "007",
		"statusdata": "Y",
	}

	authToken := "NDS/SVCLOC/TOKEN"

	switch string(fields["svc"]) {
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
