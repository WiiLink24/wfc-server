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

func handleFileRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request,
	fileRequest FileRequest) {

	gameIdString := request.URL.Query().Get("gameid")
	gameId, err := strconv.Atoi(gameIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid GameSpy game id")
		return
	}

	var handler func(string, http.ResponseWriter, *http.Request)
	var handlerExists bool
	switch fileRequest {
	case FileRequestDownload:
		handler, handlerExists = fileDownloadHandlers[gameId]
	case FileRequestUpload:
		handler, handlerExists = fileUploadHandlers[gameId]
	default:
		logging.Error(moduleName, "Invalid file request")
		return
	}

	if !handlerExists {
		logging.Warn(moduleName, "Unhandled file request for GameSpy game id:", aurora.Cyan(gameId))
		return
	}

	handler(moduleName, responseWriter, request)
}
