package sake

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type playerInfo struct {
	MiiData      common.Mii // 0x00
	ControllerId byte       // 0x4C
	Unknown      byte       // 0x4D
	StateCode    byte       // 0x4E
	CountryCode  byte       // 0x4F
}

const (
	playerInfoSize = 0x50

	rkgdFileName = "ghost.bin"
)

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
	}

	// ***********************************************************************************************************
	//	PiporGames patch for small DBs
	//
	//  Set max nº of increments for time to do when searching (setting a random value might help with ghost variety)
	increments := rand.Intn(20) + 1
	//
	//	Set debug messages (setting to false will prevent spamming the console when failing whole courseIds or time increments iterations.
	detailedDebugLog := true
	// ***********************************************************************************************************

	// try vanilla behaviour first
	ghost, err := database.GetMarioKartWiiGhostFile(pool, ctx, courseId, time, pid)
	if err != nil {
		logging.Error(moduleName, "Failed to get a ghost file from the database:", err)
		logging.Warn(moduleName, "courseId request failed, testing with random courseIds and incremental times.")
		responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultServerError))

		// we failed. choose a random courseId one.
		// choose courseId max (vanilla) up to 31
		allCourses := make([]int, 32) // 32 para incluir el 31
		for i := 0; i < 32; i++ {
			allCourses[i] = i
		}

		// Random shuffle
		rand.Shuffle(len(allCourses), func(i, j int) { allCourses[i], allCourses[j] = allCourses[j], allCourses[i] })

		// increments for up to N times until reaching max score.
		// max score is 360000 (noted in above code)
		timeqntAdd := (360000 - time) / increments
		for time < 360000 {

			// Cycle all courseIds
			for _, courseIdranInt := range allCourses {

				//logging.Info(moduleName, "Testing with", courseIdranInt, "...")

				// Test that the courseId chosen is good and convert it.
				courseIdran := common.MarioKartWiiCourseId(courseIdranInt)
				if !courseId.IsValid() {
					logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseIdranInt))
					responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultMissingParameter))
					break // we cannot continue, this is not normal behaviour.
				}

				// If courseId is good, download.
				ghost, err = database.GetMarioKartWiiGhostFile(pool, ctx, courseIdran, time, pid)
				if err != nil || len(ghost) <= 0 {
					continue // ghost cannot be found or invalid, continue.
				}

				// no errors, we are good!
				logging.Info(moduleName, "Valid ghost found with randomized courseId:", courseIdranInt, ", time:", time)
				break
			}

			// check if we run out of courseIds
			if err != nil {
				if detailedDebugLog == true {
					logging.Error(moduleName, "No courseIds left, no ghost matches criteria:", err)
					responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultServerError))
					logging.Warn(moduleName, "Retrying with a higher time:", time)
				}
				time = time + timeqntAdd // try another time
			} else {
				break
			}
		}

		// check if we run out of time increments
		if err != nil {
			logging.Error(moduleName, "No time increments left, no ghost matches criteria:", err)
			responseWriter.Header().Set(SakeFileResultHeader, strconv.Itoa(SakeFileResultServerError))
			return
		}

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

	if !common.RKGhostData(ghostFile).IsRKGDFileValid(moduleName, courseId, score) {
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

	if playerInfo.MiiData.RFLCalculateCRC() != 0x0000 {
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
