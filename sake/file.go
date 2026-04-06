package sake

import (
	"net/http"
	"strconv"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

const (
	FileRequestDownload = iota
	FileRequestUpload
)

type FileRequest int

var fileDownloadHandlers = map[int]func(string, http.ResponseWriter, *http.Request){
	common.GetGameIDOrPanic("mariokartwii"): handleMarioKartWiiFileDownloadRequest,
}

var fileUploadHandlers = map[int]func(string, http.ResponseWriter, *http.Request){
	common.GetGameIDOrPanic("mariokartwii"): handleMarioKartWiiFileUploadRequest,
}

func handleFileDownloadRequest(w http.ResponseWriter, r *http.Request) {
	moduleName := "SAKE:File:" + r.RemoteAddr

	gameIdString := r.URL.Query().Get("gameid")
	gameId, err := strconv.Atoi(gameIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid GameSpy game ID:", aurora.Cyan(gameIdString))
		return
	}

	handler, handlerExists := fileDownloadHandlers[gameId]
	if !handlerExists {
		logging.Warn(moduleName, "Unhandled file download request for GameSpy game ID:", aurora.Cyan(gameId))
		return
	}

	handler(moduleName, w, r)
}

func handleFileUploadRequest(w http.ResponseWriter, r *http.Request) {
	moduleName := "SAKE:File:" + r.RemoteAddr

	gameIdString := r.URL.Query().Get("gameid")
	gameId, err := strconv.Atoi(gameIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid GameSpy game ID:", aurora.Cyan(gameIdString))
		return
	}

	handler, handlerExists := fileUploadHandlers[gameId]
	if !handlerExists {
		logging.Warn(moduleName, "Unhandled file upload request for GameSpy game ID:", aurora.Cyan(gameId))
		return
	}

	handler(moduleName, w, r)
}
