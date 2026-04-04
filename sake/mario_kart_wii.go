package sake

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type playerInfo struct {
	MiiData      common.RawMii // 0x00
	ControllerId byte          // 0x4C
	Unknown      byte          // 0x4D
	StateCode    byte          // 0x4E
	CountryCode  byte          // 0x4F
}

const (
	playerInfoSize = 0x50

	rkgdFileName = "ghost.bin"
)

var (
	ghostDataFilterRegex = regexp.MustCompile(`^course = ([1-9]\d?|0) and gameid = 1687 and time < ([1-9][0-9]{0,5})$`)
)

func getMarioKartWiiGhostDataRecord(moduleName string, request StorageRequestData) ([]database.SakeRecord, bool) {
	if request.Sort != "time desc" {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid sort string:", aurora.Cyan(request.Sort))
		return []database.SakeRecord{}, false
	}

	if request.Offset != 0 {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid offset value:", aurora.Cyan(request.Offset))
		return []database.SakeRecord{}, false
	}

	if request.Max != 1 {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid number of records to return:", aurora.Cyan(request.Max))
		return []database.SakeRecord{}, false
	}

	if request.Surrounding != 0 {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid number of surrounding records to return:", aurora.Cyan(request.Surrounding))
		return []database.SakeRecord{}, false
	}

	if len(request.OwnerIDs.OwnerID) != 0 {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid owner id array:", aurora.Cyan(request.OwnerIDs))
		return []database.SakeRecord{}, false
	}

	if request.CacheFlag != 0 {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid cache value:", aurora.Cyan(request.CacheFlag))
		return []database.SakeRecord{}, false
	}

	match := ghostDataFilterRegex.FindStringSubmatch(request.Filter)
	if match == nil {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid filter string:", aurora.Cyan(request.Filter))
		return []database.SakeRecord{}, false
	}

	courseIdInt, _ := strconv.Atoi(match[1])
	courseId := common.MarioKartWiiCourseId(courseIdInt)
	if !courseId.IsValid() {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid course ID:", aurora.Cyan(match[1]))
		return []database.SakeRecord{}, false
	}

	time, _ := strconv.Atoi(match[2])
	if time >= 360000 /* 6 minutes */ {
		logging.Error(moduleName, "mariokartwii/GhostData: Invalid time:", aurora.Cyan(match[2]))
		return []database.SakeRecord{}, false
	}

	fileId, err := database.GetMarioKartWiiGhostData(pool, ctx, courseId, time)
	if err != nil {
		logging.Error(moduleName, "mariokartwii/GhostData: Failed to get the ghost data from the database:", err)
		return []database.SakeRecord{}, false
	}

	return []database.SakeRecord{{
		GameId:   1687,
		TableId:  "GhostData",
		RecordId: 0,
		OwnerId:  0,
		Fields: map[string]database.SakeField{
			"fileid": {
				Type:  database.SakeFieldTypeInt,
				Value: strconv.FormatInt(int64(int32(fileId)), 10),
			},
		},
	}}, true
}

func getMarioKartWiiStoredGhostDataRecord(moduleName string, request StorageRequestData) ([]database.SakeRecord, bool) {
	if request.Sort != "time" {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Invalid sort string:", aurora.Cyan(request.Sort))
		return []database.SakeRecord{}, false
	}

	if request.Offset != 0 {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Invalid offset value:", aurora.Cyan(request.Offset))
		return []database.SakeRecord{}, false
	}

	if request.Max != 1 {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Invalid number of records to return:", aurora.Cyan(request.Max))
		return []database.SakeRecord{}, false
	}

	if request.Surrounding != 0 {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Invalid number of surrounding records to return:", aurora.Cyan(request.Surrounding))
		return []database.SakeRecord{}, false
	}

	if len(request.OwnerIDs.OwnerID) != 0 {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Invalid owner id array:", aurora.Cyan(request.OwnerIDs))
		return []database.SakeRecord{}, false
	}

	if request.CacheFlag != 0 {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Invalid cache value:", aurora.Cyan(request.CacheFlag))
		return []database.SakeRecord{}, false
	}

	match := regexp.MustCompile(`^course = ([1-9]\d?|0) and gameid = 1687(?: and region = ([1-7]))?$`).FindStringSubmatch(request.Filter)
	if match == nil {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Invalid filter string:", aurora.Cyan(request.Filter))
		return []database.SakeRecord{}, false
	}

	courseIdInt, _ := strconv.Atoi(match[1])
	courseId := common.MarioKartWiiCourseId(courseIdInt)
	if !courseId.IsValid() {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Invalid course ID:", aurora.Cyan(match[1]))
		return []database.SakeRecord{}, false
	}

	var regionId common.MarioKartWiiLeaderboardRegionId
	if regionIdExists := match[2] != ""; regionIdExists {
		regionIdInt, _ := strconv.Atoi(match[2])
		regionId = common.MarioKartWiiLeaderboardRegionId(regionIdInt)
	} else {
		regionId = common.Worldwide
	}

	pid, fileId, err := database.GetMarioKartWiiStoredGhostData(pool, ctx, regionId, courseId)
	if err != nil {
		logging.Error(moduleName, "mariokartwii/StoredGhostData: Failed to get the stored ghost data from the database:", err)
		return []database.SakeRecord{}, false
	}

	return []database.SakeRecord{{
		GameId:   1687,
		TableId:  "StoredGhostData",
		RecordId: 0,
		OwnerId:  int32(pid),
		Fields: map[string]database.SakeField{
			"profile": {Type: database.SakeFieldTypeInt, Value: strconv.FormatInt(int64(int32(pid)), 10)},
			"fileid":  {Type: database.SakeFieldTypeInt, Value: strconv.FormatInt(int64(int32(fileId)), 10)},
		},
	}}, true
}

func handleMarioKartWiiFileDownloadRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	if strings.HasSuffix(request.URL.Path, "ghostdownload.aspx") {
		handleMarioKartWiiGhostDownloadRequest(moduleName, responseWriter, request)
		return
	}

	query := request.URL.Query()

	fileIdString := query.Get("fileid")
	pidString := query.Get("pid")

	fileId, err := strconv.Atoi(fileIdString)
	if err != nil || fileId <= 0 {
		logging.Error(moduleName, "Invalid file ID:", aurora.Cyan(fileIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	pid, err := strconv.Atoi(pidString)
	if err != nil || pid <= 0 {
		logging.Error(moduleName, "Invalid profile ID:", aurora.Cyan(pidString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	file, err := database.GetMarioKartWiiFile(pool, ctx, fileId)
	if err != nil {
		logging.Error(moduleName, "Failed to get the file from the database:", err)
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultServerError))
		return
	}

	responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultSuccess))
	responseWriter.Header().Set("Content-Length", strconv.Itoa(len(file)))
	responseWriter.Write(file)
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
	if err != nil || time <= 0 || time >= 360000 /* 6 minutes */ {
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

	ghostData := common.RKGhostData(ghost)
	ghostData.SetMiiData(ghostData.GetMiiData().ClearMiiInfo())
	ghostData.RecalculateCRC()

	responseBody := append(downloadedGhostFileHeader(), []byte(ghostData)...)

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
	if courseId < common.MarioCircuit || isContest == courseId.IsValid() || courseId > 32767 {
		logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseIdString))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}

	score, err := strconv.Atoi(scoreString)
	if err != nil || score <= 0 || score >= 360000 /* 6 minutes */ {
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

	fixedPlayerInfo, ok := isPlayerInfoValid(playerInfo)
	if !ok {
		logging.Error(moduleName, "Invalid player info:", aurora.Cyan(playerInfo))
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
		return
	}
	playerInfo = fixedPlayerInfo

	// The multipart boundary utilized by GameSpy does not conform to RFC 2045. To ensure compliance,
	// we need to surround it with double quotation marks.
	contentType := request.Header.Get("Content-Type")
	boundary := getMultipartBoundary(contentType)
	if boundary == GameSpyMultipartBoundary {
		quotedBoundary := fmt.Sprintf("%q", boundary)
		contentType := strings.Replace(contentType, boundary, quotedBoundary, 1)
		request.Header.Set("Content-Type", contentType)
	}

	err = request.ParseMultipartForm(common.RKGDFileMaxSize)
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

	if fileHeader.Size < common.RKGDFileMinSize || fileHeader.Size > common.RKGDFileMaxSize {
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

	ghostData := common.RKGhostData(ghostFile)
	if !ghostData.IsRKGDFileValid(moduleName, courseId, score) {
		logging.Error(moduleName, "Received an invalid ghost file")
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultFileTooLarge))
		return
	}

	ghostData.SetMiiData(ghostData.GetMiiData().ClearMiiInfo())
	ghostData.RecalculateCRC()

	if isContest {
		ghostFile = nil
	}

	err = database.InsertMarioKartWiiGhostFile(pool, ctx, regionId, courseId, score, pid, playerInfo, []byte(ghostData))
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

// isPlayerInfoValid checks if the player info (base64.URLEncoding) string is valid, and if so, returns a "fixed" version of it with base64.StdEncoding
func isPlayerInfoValid(playerInfoString string) (string, bool) {
	playerInfoByteArray, err := base64.URLEncoding.DecodeString(playerInfoString)
	if err != nil {
		return "", false
	}

	if len(playerInfoByteArray) != playerInfoSize {
		return "", false
	}

	var playerInfo playerInfo
	reader := bytes.NewReader(playerInfoByteArray)
	err = binary.Read(reader, binary.BigEndian, &playerInfo)
	if err != nil {
		return "", false
	}

	if playerInfo.MiiData.CalculateMiiCRC() != 0x0000 {
		return "", false
	}

	controllerId := common.MarioKartWiiControllerId(playerInfo.ControllerId)

	if !controllerId.IsValid() {
		return "", false
	}

	playerInfo.MiiData.ClearMiiInfo()

	fixedPlayerInfoByteArray := new(bytes.Buffer)
	err = binary.Write(fixedPlayerInfoByteArray, binary.BigEndian, playerInfo)
	if err != nil {
		return "", false
	}

	fixedPlayerInfoString := base64.StdEncoding.EncodeToString(fixedPlayerInfoByteArray.Bytes())
	return fixedPlayerInfoString, true
}

func getMultipartBoundary(contentType string) string {
	startIndex := strings.Index(contentType, "boundary=")
	if startIndex == -1 {
		return ""
	}
	startIndex += len("boundary=")

	return contentType[startIndex:]
}
