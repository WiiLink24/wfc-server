package sake

import (
	"github.com/logrusorgru/aurora/v3"
	"net/http"
	"wwfc/logging"
)

func HandleRequest(w http.ResponseWriter, r *http.Request) {
	logging.Notice("SAKE", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))

	switch r.URL.String() {
	case "/SakeStorageServer/StorageServer.asmx":
		moduleName := "SAKE:Storage:" + r.RemoteAddr
		handleStorageRequest(moduleName, w, r)
		break
	}
}
