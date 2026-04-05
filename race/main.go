package race

import (
	"net/http"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

var (
	db database.Connection
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	common.ReadGameList()

	// Start SQL
	db = database.Start(config)
}

func Shutdown() {
	db.Close()
}

func HandleRequest(responseWriter http.ResponseWriter, request *http.Request) {
	logging.Info("RACE", aurora.Yellow(request.Method), aurora.Cyan(request.URL), "via", aurora.Cyan(request.Host), "from", aurora.BrightCyan(request.RemoteAddr))

	switch {
	case strings.HasSuffix(request.URL.Path, "NintendoRacingService.asmx"):
		moduleName := "RACE:RacingService:" + request.RemoteAddr
		handleNintendoRacingServiceRequest(moduleName, responseWriter, request)
	}
}
