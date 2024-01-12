package gpsp

import (
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func handleOthersList(command common.GameSpyCommand) string {
	moduleName := "GPSP"

	strProfileId, ok := command.OtherValues["profileid"]
	if !ok {
		logging.Error(moduleName, "Missing profileid in otherslist")
		return gpcm.ErrSearch.GetMessage()
	}

	profileId, err := strconv.ParseUint(strProfileId, 10, 32)
	if err != nil {
		logging.Error(moduleName, "Invalid profileid:", strProfileId)
		return gpcm.ErrSearch.GetMessage()
	}

	moduleName = "GPSP:" + strconv.FormatUint(profileId, 10)
	logging.Info(moduleName, "Lookup otherslist for", aurora.Cyan(profileId))

	strSessionKey, ok := command.OtherValues["sesskey"]
	if !ok {
		logging.Error(moduleName, "Missing sesskey in otherslist")
		return gpcm.ErrSearch.GetMessage()
	}

	sessionKey, err := strconv.ParseInt(strSessionKey, 10, 32)
	if err != nil {
		logging.Error(moduleName, "Invalid sesskey:", strSessionKey)
		return gpcm.ErrSearch.GetMessage()
	}

	numopids, ok := command.OtherValues["numopids"]
	if !ok {
		logging.Error(moduleName, "Missing numopids in otherslist")
		return gpcm.ErrSearch.GetMessage()
	}

	opids, ok := command.OtherValues["opids"]
	if !ok {
		logging.Error(moduleName, "Missing opids in otherslist")
		return gpcm.ErrSearch.GetMessage()
	}

	gameName, ok := command.OtherValues["gamename"]
	if !ok {
		logging.Error(moduleName, "Missing gamename in otherslist")
		return gpcm.ErrSearch.GetMessage()
	}

	numOpidsValue, err := strconv.Atoi(numopids)
	if err != nil {
		logging.Error(moduleName, "Invalid numopids:", numopids)
		return gpcm.ErrSearch.GetMessage()
	}

	var opidsSplit []string
	if strings.Contains(opids, "|") {
		opidsSplit = strings.Split(opids, "|")
	} else if opids != "" && opids != "0" {
		opidsSplit = append(opidsSplit, opids)
	}

	if len(opidsSplit) != numOpidsValue && opids != "0" {
		logging.Error(moduleName, "Mismatch opids length with numopids:", aurora.Cyan(len(opidsSplit)), "!=", aurora.Cyan(numOpidsValue))
		return gpcm.ErrSearch.GetMessage()
	}

	// Lookup profile ID using GPCM
	uniqueNick, ok := gpcm.VerifyPlayerSearch(uint32(profileId), int32(sessionKey), gameName)
	if !ok {
		logging.Error(moduleName, "otherslist verify failed")
		return gpcm.ErrSearch.GetMessage()
	}

	payload := `\otherslist\`
	for _, strOtherId := range opidsSplit {
		payload += `\o\` + strOtherId
		payload += `\uniquenick\` + uniqueNick
	}

	payload += `\oldone\\final\`
	return payload
}
