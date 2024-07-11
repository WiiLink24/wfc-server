package race

import (
	"net/http"
	"path"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

func StartServer(reload bool) {
}

func Shutdown() {
}

func HandleRequest(responseWriter http.ResponseWriter, request *http.Request) {
	logging.Info("RACE", aurora.Yellow(request.Method), aurora.Cyan(request.URL), "via", aurora.Cyan(request.Host), "from", aurora.BrightCyan(request.RemoteAddr))

	switch path.Base(request.URL.Path) {
	case "NintendoRacingService.asmx":
		moduleName := "RACE:RacingService:" + request.RemoteAddr
		handleNintendoRacingServiceRequest(moduleName, responseWriter, request)
	}
}
