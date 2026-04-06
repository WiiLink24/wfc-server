package gpsp

import (
	"wwfc/common"
	"wwfc/gpcm"
	"wwfc/logging"
)

var ServerName = "gpsp"

func StartServer(reload bool) {
}

func Shutdown() {
}

func NewConnection(index uint64, address string) {
}

func CloseConnection(index uint64) {
}

func HandlePacket(index uint64, data []byte) {
	moduleName := "GPSP"

	// TODO: Handle split packets
	message := ""
	for _, b := range data {
		message += string(b)
	}

	commands, err := common.ParseGameSpyMessage(message)
	if err != nil {
		logging.Error(moduleName, "Error parsing message:", err.Error())
		logging.Error(moduleName, "Raw data:", message)
		replyError(moduleName, index, gpcm.ErrParse)
		return
	}

	for _, command := range commands {
		switch command.Command {
		default:
			logging.Error(moduleName, "Unknown command:", command.Command)
			logging.Error(moduleName, "Raw data:", message)
			replyError(moduleName, index, gpcm.ErrParse)

		case "ka":
			err = common.SendPacket(ServerName, index, []byte(`\ka\\final\`))

		case "otherslist":
			err = common.SendPacket(ServerName, index, []byte(handleOthersList(command)))

		case "search":
			err = common.SendPacket(ServerName, index, []byte(handleSearch(command)))
		}
	}
	if err != nil {
		logging.Error(moduleName, "Failed to send packet:", err)
	}
}
