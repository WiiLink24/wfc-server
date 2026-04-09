package nas

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

var (
	dlcDir = "./dlc"
)

func parseAuthRequest(r *http.Request) (map[string][]byte, error) {
	moduleName := getModuleName(r)

	err := r.ParseForm()
	if err != nil {
		logging.Error(moduleName, "Failed to parse form")
		return nil, errors.New("failed to parse form")
	}

	// Need to know this here to determine UTF-16 endianness (LE for DS, BE for Wii)
	// unitcd 0 = DS, 1 = Wii
	isWii := false
	if unitcdValues, ok := r.PostForm["unitcd"]; ok {
		unitcdDecoded, err := common.Base64DwcEncoding.DecodeString(unitcdValues[0])
		if err != nil {
			logging.Error(moduleName, "Invalid unitcd string in form")
			return nil, errors.New("invalid unitcd string in form")
		}
		isWii = len(unitcdDecoded) != 1 || unitcdDecoded[0] != '0'
	}

	var endianness binary.ByteOrder = binary.LittleEndian
	if isWii {
		endianness = binary.BigEndian
	}

	fields := map[string][]byte{}
	for key, values := range r.PostForm {
		if len(values) != 1 {
			logging.Warn(moduleName, "Ignoring none or multiple POST form values:", aurora.Cyan(key).String()+":", aurora.Cyan(values))
			continue
		}

		if strings.HasPrefix(key, "_") {
			// Values unique to CTGP/the Wiimmfi payload. Ignored for compatibility reasons.
			continue
		}

		parsed, err := common.Base64DwcEncoding.DecodeString(values[0])
		if err != nil {
			logging.Error(moduleName, "Invalid POST form value:", aurora.Cyan(key).String()+":", aurora.Cyan(values[0]))
			return nil, errors.New("invalid POST form value: " + key)
		}

		fields[key] = parsed

		reported := string(parsed)
		if key == "ingamesn" || key == "devname" || key == "words" {
			// Special handling required for reporting the UTF-16 strings
			reported = common.UTF16Decode(parsed, endianness)
		}
		logging.Info(moduleName, aurora.Cyan(key).String()+":", aurora.Cyan(reported))
	}

	return fields, nil
}

func writeAuthResponse(w http.ResponseWriter, reply map[string]string) {
	var response []byte
	param := url.Values{}
	for key, value := range reply {
		param.Set(key, common.Base64DwcEncoding.EncodeToString([]byte(value)))
	}
	response = []byte(param.Encode())
	response = []byte(strings.ReplaceAll(string(response), "%2A", "*"))

	// DWC treats the response like a null terminated string
	response = append(response, 0x00)

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	_, err := w.Write(response)
	if err != nil {
		logging.Error("NAS", "Error writing response:", err)
	}
}

func getDateTime() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
