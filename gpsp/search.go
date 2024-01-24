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
	logging.Info(moduleName, "Search for", aurora.Cyan(profileId))

	return common.CreateGameSpyMessage(common.GameSpyCommand{
		Command: "bsrdone",
	})
}
