package gpsp

import (
	"strconv"
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func handleSearch(command common.GameSpyCommand) string {
	moduleName := "GPSP"

	strProfileId, ok := command.OtherValues["profileid"]
	if !ok {
		logging.Error(moduleName, "Missing profileid in search")
		return gpcm.ErrSearch.GetMessage()
	}

	profileId, err := strconv.ParseUint(strProfileId, 10, 32)
	if err != nil {
		logging.Error(moduleName, "Invalid profileid:", strProfileId)
		return gpcm.ErrSearch.GetMessage()
	}

	moduleName = "GPSP:" + strconv.FormatUint(profileId, 10)

	logInfo := ""
	for _, field := range []string{
		"nick", "uniquenick", "email", "firstname", "lastname", "icquin", "skip",
	} {
		if value, ok := command.OtherValues[field]; ok {
			logInfo += " " + aurora.BrightCyan(field).String() + ": '" + aurora.Cyan(value).String() + "'"
		}
	}

	if logInfo == "" {
		logging.Info(moduleName, "Search with no fields")
	} else {
		logging.Info(moduleName, "Search"+logInfo)
	}

	return common.CreateGameSpyMessage(common.GameSpyCommand{
		Command: "bsrdone",
	})
}
