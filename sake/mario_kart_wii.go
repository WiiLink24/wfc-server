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

func handleMarioKartWiiFileUploadRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()

	regionIdString := query.Get("regionid")
	courseIdString := query.Get("courseid")
	scoreString := query.Get("score")
	pidString := query.Get("pid")
	playerInfo := query.Get("playerinfo")

	regionIdInt, err := strconv.Atoi(regionIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid region id")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultMissingParameter))
		return
	}
	regionId := common.MarioKartWiiLeaderboardRegionId(regionIdInt)
	if !regionId.IsValid() || regionId == common.Worldwide {
		logging.Error(moduleName, "Invalid region id")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultMissingParameter))
		return
	}

	courseIdInt, err := strconv.Atoi(courseIdString)
	if err != nil {
		logging.Error(moduleName, "Invalid course id")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultMissingParameter))
		return
	}
	courseId := common.MarioKartWiiCourseId(courseIdInt)
	if courseId < common.MarioCircuit || courseId > 32767 {
		logging.Error(moduleName, "Invalid course id")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultMissingParameter))
		return
	}

	score, err := strconv.Atoi(scoreString)
	if err != nil || score <= 0 {
		logging.Error(moduleName, "Invalid score")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultMissingParameter))
		return
	}

	pid, err := strconv.Atoi(pidString)
	if err != nil || pid <= 0 {
		logging.Error(moduleName, "Invalid pid")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultMissingParameter))
		return
	}

	if !isPlayerInfoValid(playerInfo) {
		logging.Error(moduleName, "Invalid player info")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultMissingParameter))
		return
	}
	// Mario Kart Wii expects player information to be in this form
	playerInfo, _ = common.GameSpyBase64ToBase64(playerInfo, common.GameSpyBase64EncodingURLSafe)

	// The multipart boundary utilized by GameSpy does not conform to RFC 2045. To ensure compliance,
	// we need to surround it with double quotation marks.
	contentType := request.Header.Get("Content-Type")
	boundary := getMultipartBoundary(contentType)
	if boundary == common.GameSpyMultipartBoundary {
		quotedBoundary := fmt.Sprintf("%q", boundary)
		contentType := strings.Replace(contentType, boundary, quotedBoundary, 1)
		request.Header.Set("Content-Type", contentType)
	}

	err = request.ParseMultipartForm(rkgdFileMaxSize)
	if err != nil {
		logging.Error(moduleName, "Failed to parse the multipart form")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultFileNotFound))
		return
	}

	file, fileHeader, err := request.FormFile(rkgdFileName)
	if err != nil {
		logging.Error(moduleName, "Failed to find the ghost file")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultFileNotFound))
		return
	}
	defer file.Close()

	if fileHeader.Size < rkgdFileMinSize || fileHeader.Size > rkgdFileMaxSize {
		logging.Error(moduleName, "The size of the ghost file is invalid")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultFileTooLarge))
		return
	}

	ghostFile := make([]byte, fileHeader.Size)
	_, err = io.ReadFull(file, ghostFile)
	if err != nil {
		logging.Error(moduleName, "Failed to read contents of the ghost file")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultFileTooLarge))
		return
	}

	if !isRKGDFileValid(ghostFile) {
		logging.Error(moduleName, "Received an invalid ghost file")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultFileTooLarge))
		return
	}

	err = database.UploadMarioKartWiiGhostFile(pool, ctx, regionId, courseId, score, pid, playerInfo, ghostFile)
	if err != nil {
		logging.Error(moduleName, "Failed to insert the ghost file into the database")
		responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultServerError))
		return
	}

	responseWriter.Header().Set(common.SakeFileResultHeader, strconv.Itoa(common.SakeFileResultSuccess))
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
