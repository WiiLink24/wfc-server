package sake

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type playerInfo struct {
	MiiData      [0x4C]byte // 0x00
	ControllerId byte       // 0x4C
	Unknown      byte       // 0x4D
	StateCode    byte       // 0x4E
	CountryCode  byte       // 0x4F
}

const (
	playerInfoSize = 0x50

	rkgdFileMaxSize = 0x2800
	rkgdFileMinSize = 0x0088 + 0x0008 + 0x0004
	rkgdFileName    = "ghost.bin"
)

func handleMarioKartWiiFileDownloadRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	if strings.HasSuffix(request.URL.Path, "ghostdownload.aspx") {
		handleMarioKartWiiGhostDownloadRequest(moduleName, responseWriter, request)
		return
	}
}

func handleMarioKartWiiGhostDownloadRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()

	regionIdString := query.Get("region")
	pidString := query.Get("p0")
	courseIdString := query.Get("c0")
	timeString := query.Get("t0")

	regionIdInt, err := strconv.Atoi(regionIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid region ID:", aurora.Cyan(regionIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}
	if common.MarioKartWiiLeaderboardRegionId(regionIdInt) != common.Worldwide {
		logging.Error(moduleName, "Invalid region ID:", aurora.Cyan(regionIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	courseIdInt, err := strconv.Atoi(courseIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}
	courseId := common.MarioKartWiiCourseId(courseIdInt)
	if !courseId.IsValid() {
		logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	pid, err := strconv.Atoi(pidString)
	if err != nil || pid <= 0 {
		logging.Error(moduleName, "Invalid profile ID:", aurora.Cyan(pidString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	time, err := strconv.Atoi(timeString)
	if err != nil || time <= 0 {
		logging.Error(moduleName, "Invalid time:", aurora.Cyan(timeString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	ghost, err := database.GetMarioKartWiiGhostFile(pool, ctx, courseId, time, pid)
	if err != nil {
		logging.Error(moduleName, "Failed to get a ghost file from the database:", err)
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultServerError))
		return
	}

	responseBody := append(downloadedGhostFileHeader(), ghost...)

	responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultSuccess))
	responseWriter.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	responseWriter.Write(responseBody)
}

func handleMarioKartWiiFileUploadRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	if strings.HasSuffix(request.URL.Path, "ghostupload.aspx") {
		handleMarioKartWiiGhostUploadRequest(moduleName, responseWriter, request)
		return
	}
}

func handleMarioKartWiiGhostUploadRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()

	regionIdString := query.Get("regionid")
	courseIdString := query.Get("courseid")
	scoreString := query.Get("score")
	pidString := query.Get("pid")
	playerInfo := query.Get("playerinfo")
	_, isContest := query["contest"]

	regionIdInt, err := strconv.Atoi(regionIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid region ID:", aurora.Cyan(regionIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}
	regionId := common.MarioKartWiiLeaderboardRegionId(regionIdInt)
	if !regionId.IsValid() || regionId == common.Worldwide {
		logging.Error(moduleName, "Invalid region ID:", aurora.Cyan(regionIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	courseIdInt, err := strconv.Atoi(courseIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}
	courseId := common.MarioKartWiiCourseId(courseIdInt)
	if courseId < common.MarioCircuit || courseId > 32767 {
		logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	score, err := strconv.Atoi(scoreString)
	if err != nil || score <= 0 {
		logging.Error(moduleName, "Invalid score:", aurora.Cyan(scoreString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	pid, err := strconv.Atoi(pidString)
	if err != nil || pid <= 0 {
		logging.Error(moduleName, "Invalid profile ID:", aurora.Cyan(pidString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	if !isPlayerInfoValid(playerInfo) {
		logging.Error(moduleName, "Invalid player info:", aurora.Cyan(playerInfo))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}
	// Mario Kart Wii expects player information to be in this form
	playerInfo, _ = common.GameSpyBase64ToBase64(playerInfo, common.GameSpyBase64EncodingURLSafe)

	// The multipart boundary utilized by GameSpy does not conform to RFC 2045. To ensure compliance,
	// we need to surround it with double quotation marks.
	contentType := request.Header.Get("Content-Type")
	boundary := getMultipartBoundary(contentType)
	if boundary == GameSpyMultipartBoundary {
		quotedBoundary := fmt.Sprintf("%q", boundary)
		contentType := strings.Replace(contentType, boundary, quotedBoundary, 1)
		request.Header.Set("Content-Type", contentType)
	}

	err = request.ParseMultipartForm(rkgdFileMaxSize)
	if err != nil {
		logging.Error(moduleName, "Failed to parse the multipart form:", err)
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultFileNotFound))
		return
	}

	file, fileHeader, err := request.FormFile(rkgdFileName)
	if err != nil {
		logging.Error(moduleName, "Failed to find the ghost file:", err)
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultFileNotFound))
		return
	}
	defer file.Close()

	if fileHeader.Size < rkgdFileMinSize || fileHeader.Size > rkgdFileMaxSize {
		logging.Error(moduleName, "The size of the ghost file is invalid:", aurora.Cyan(fileHeader.Size))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultFileTooLarge))
		return
	}

	ghostFile := make([]byte, fileHeader.Size)
	_, err = io.ReadFull(file, ghostFile)
	if err != nil {
		logging.Error(moduleName, "Failed to read contents of the ghost file:", err)
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultFileTooLarge))
		return
	}

	if !isRKGDFileValid(ghostFile) {
		logging.Error(moduleName, "Received an invalid ghost file")
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultFileTooLarge))
		return
	}

	if isContest {
		ghostFile = nil
	}

	err = database.InsertMarioKartWiiGhostFile(pool, ctx, regionId, courseId, score, pid, playerInfo, ghostFile)
	if err != nil {
		logging.Error(moduleName, "Failed to insert the ghost file into the database:", err)
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultServerError))
		return
	}

	responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultSuccess))
}

func downloadedGhostFileHeader() []byte {
	var downloadedGhostFileHeader [0x200]byte

	binary.BigEndian.PutUint32(downloadedGhostFileHeader[0x40:0x44], uint32(len(downloadedGhostFileHeader)))

	return downloadedGhostFileHeader[:]
}

func isPlayerInfoValid(playerInfoString string) bool {
	playerInfoByteArray, err := common.DecodeGameSpyBase64(playerInfoString, common.GameSpyBase64EncodingURLSafe)
	if err != nil {
		return false
	}

	if len(playerInfoByteArray) != playerInfoSize {
		return false
	}

	var playerInfo playerInfo
	reader := bytes.NewReader(playerInfoByteArray)
	err = binary.Read(reader, binary.BigEndian, &playerInfo)
	if err != nil {
		return false
	}

	if common.RFLCalculateCRC(playerInfo.MiiData[:]) != 0x0000 {
		return false
	}

	controllerId := common.MarioKartWiiControllerId(playerInfo.ControllerId)

	return controllerId.IsValid()
}

func getMultipartBoundary(contentType string) string {
	startIndex := strings.Index(contentType, "boundary=")
	if startIndex == -1 {
		return ""
	}
	startIndex += len("boundary=")

	return contentType[startIndex:]
}

func isRKGDFileValid(rkgdFile []byte) bool {
	rkgdFileMagic := []byte{'R', 'K', 'G', 'D'}

	if !bytes.Equal(rkgdFile[:4], rkgdFileMagic) {
		return false
	}

	rkgdFileLength := len(rkgdFile)

	expectedChecksum := binary.BigEndian.Uint32(rkgdFile[rkgdFileLength-4:])
	checksum := crc32.ChecksumIEEE(rkgdFile[:rkgdFileLength-4])

	return checksum == expectedChecksum
}
