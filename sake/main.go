package sake

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

func HandleRequest(w http.ResponseWriter, r *http.Request) {
	logging.Info("SAKE", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))

	urlPath := r.URL.Path
	switch {
	case urlPath == "/SakeStorageServer/StorageServer.asmx":
		moduleName := "SAKE:Storage:" + r.RemoteAddr
		handleStorageRequest(moduleName, w, r)
	case strings.HasSuffix(urlPath, "download.aspx"):
		moduleName := "SAKE:File:" + r.RemoteAddr
		handleFileRequest(moduleName, w, r, FileRequestDownload)
	case strings.HasSuffix(urlPath, "upload.aspx"):
		moduleName := "SAKE:File:" + r.RemoteAddr
		handleFileRequest(moduleName, w, r, FileRequestUpload)
	}
}
