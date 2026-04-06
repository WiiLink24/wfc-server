package sake

import (
	"net/http"
	"wwfc/common"
	"wwfc/database"
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

func RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /SakeStorageServer/StorageServer.asmx", handleStorageRequest)
	mux.HandleFunc("GET /SakeFileServer/download.aspx", handleFileDownloadRequest)
	mux.HandleFunc("POST /SakeFileServer/upload.aspx", handleFileUploadRequest)
	mux.HandleFunc("GET /SakeFileServer/ghostdownload.aspx", handleMarioKartWiiGhostDownloadRequest)
}
