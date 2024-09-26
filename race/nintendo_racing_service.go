package race

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type rankingsRequestEnvelope struct {
	Body rankingsRequestBody `xml:"Body"`
}

type rankingsRequestBody struct {
	GetTopTenRankings rankingsRequestGetTopTenRankings `xml:"GetTopTenRankings"`
}

type rankingsRequestGetTopTenRankings struct {
	GameId   int                                    `xml:"gameid"`
	RegionId common.MarioKartWiiLeaderboardRegionId `xml:"regionid"`
	CourseId common.MarioKartWiiCourseId            `xml:"courseid"`
}

type rankingsResponseRankingDataResponse struct {
	XMLName      xml.Name          `xml:"RankingDataResponse"`
	XMLNSXSI     string            `xml:"xmlns:xsi,attr"`
	XMLNSXSD     string            `xml:"xmlns:xsd,attr"`
	XMLNS        string            `xml:"xmlns,attr"`
	ResponseCode raceServiceResult `xml:"responseCode"`
	DataArray    rankingsResponseDataArray
}

type rankingsResponseDataArray struct {
	XMLName    xml.Name `xml:"dataArray"`
	NumRecords int      `xml:"numrecords"`
	Data       []rankingsResponseData
}

type rankingsResponseData struct {
	XMLName     xml.Name `xml:"data"`
	RankingData rankingsResponseRankingData
}

type rankingsResponseRankingData struct {
	XMLName  xml.Name `xml:"RankingData"`
	OwnerID  int      `xml:"ownerid"`
	Rank     int      `xml:"rank"`
	Time     int      `xml:"time"`
	UserData string   `xml:"userdata"`
}

type submitScoresRequestEnvelope struct {
	Body submitScoresRequestBody `xml:"Body"`
}

type submitScoresRequestBody struct {
	SubmitScores submitScoresRequestSubmitScores `xml:"SubmitScores"`
}

type submitScoresRequestSubmitScores struct {
	GameData submitScoresRequestGameData `xml:"gameData"`
}

type submitScoresRequestGameData struct {
	RegionId   common.MarioKartWiiLeaderboardRegionId `xml:"regionid"`
	ProfileID  int                                    `xml:"profileid"`
	GameId     int                                    `xml:"gameid"`
	ScoreMode  scoreMode                              `xml:"scoremode"`
	ScoreDatas submitScoresRequestScoreDatas          `xml:"ScoreDatas"`
}

type submitScoresRequestScoreDatas struct {
	ScoreData []submitScoresRequestScoreData `xml:"ScoreData"`
}

type submitScoresRequestScoreData struct {
	Time             int                         `xml:"time"`
	CourseID         common.MarioKartWiiCourseId `xml:"courseid"`
	PlayerInfoBase64 string                      `xml:"playerinfobase64"`
}

type submitScoresResponse struct {
	XMLName      xml.Name          `xml:"SubmitScoresResult"`
	XMLNSXSI     string            `xml:"xmlns:xsi,attr"`
	XMLNSXSD     string            `xml:"xmlns:xsd,attr"`
	XMLNS        string            `xml:"xmlns,attr"`
	ResponseCode raceServiceResult `xml:"responseCode"`
}

type playerInfo struct {
	MiiData      common.Mii // 0x00
	ControllerId byte       // 0x4C
	Unknown      byte       // 0x4D
	StateCode    byte       // 0x4E
	CountryCode  byte       // 0x4F
}

type raceServiceResult int
type scoreMode int

const playerInfoSize = 0x50

// https://github.com/GameProgressive/UniSpySDK/blob/master/webservices/RacingService.h
const (
	raceServiceResultSuccess           = 0
	raceServiceResultDatabaseError     = 6
	raceServiceResultParseError        = 101
	raceServiceResultInvalidParameters = 105
)

const (
	scoreModeTimeTrials = iota // 0x00
	scoreModeContest           // 0x01
)

const (
	xmlNamespaceXSI = "http://www.w3.org/2001/XMLSchema-instance"
	xmlNamespaceXSD = "http://www.w3.org/2001/XMLSchema"
	xmlNamespace    = "http://gamespy.net/RaceService/"
)

var marioKartWiiGameID = common.GetGameIDOrPanic("mariokartwii") // 1687

func handleNintendoRacingServiceRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	soapActionHeader := request.Header.Get("SOAPAction")
	if soapActionHeader == "" {
		logging.Error(moduleName, "No SOAPAction header")
		return
	}

	slashIndex := strings.LastIndex(soapActionHeader, "/")
	if slashIndex == -1 {
		logging.Error(moduleName, "Invalid SOAPAction header")
		return
	}
	quotationMarkIndex := strings.Index(soapActionHeader[slashIndex+1:], "\"")
	if quotationMarkIndex == -1 {
		logging.Error(moduleName, "Invalid SOAPAction header")
		return
	}

	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		panic(err)
	}

	soapAction := soapActionHeader[slashIndex+1 : slashIndex+1+quotationMarkIndex]
	switch soapAction {
	case "GetTopTenRankings":
		handleGetTopTenRankingsRequest(moduleName, responseWriter, requestBody)
	case "SubmitScores":
		handleSubmitScoresRequest(moduleName, responseWriter, requestBody)
	}
}

func handleGetTopTenRankingsRequest(moduleName string, responseWriter http.ResponseWriter, requestBody []byte) {
	requestXML := rankingsRequestEnvelope{}
	err := xml.Unmarshal(requestBody, &requestXML)
	if err != nil {
		logging.Error(moduleName, "Got malformed XML")
		writeGetTop10RankingsResponse(raceServiceResultParseError, responseWriter, rankingsResponseDataArray{})
		return
	}

	getTopTenRankings := requestXML.Body.GetTopTenRankings

	if getTopTenRankings.GameId != marioKartWiiGameID {
		logging.Error(moduleName, "Wrong GameSpy game ID:", aurora.Cyan(getTopTenRankings.GameId))
		writeGetTop10RankingsResponse(raceServiceResultInvalidParameters, responseWriter, rankingsResponseDataArray{})
		return
	}

	regionId := getTopTenRankings.RegionId
	if !regionId.IsValid() {
		logging.Error(moduleName, "Invalid region ID:", aurora.Cyan(regionId))
		writeGetTop10RankingsResponse(raceServiceResultInvalidParameters, responseWriter, rankingsResponseDataArray{})
		return
	}
	courseId := getTopTenRankings.CourseId
	if courseId < common.MarioCircuit || courseId > 32767 {
		logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseId))
		writeGetTop10RankingsResponse(raceServiceResultInvalidParameters, responseWriter, rankingsResponseDataArray{})
		return
	}

	topTenRankings, err := database.GetMarioKartWiiTopTenRankings(pool, ctx, regionId, courseId)
	if err != nil {
		logging.Error(moduleName, "Failed to get the Top 10 rankings:", err)
		writeGetTop10RankingsResponse(raceServiceResultDatabaseError, responseWriter, rankingsResponseDataArray{})
		return
	}

	numberOfRankings := len(topTenRankings)
	data := make([]rankingsResponseData, 0, numberOfRankings)
	for i, topTenRanking := range topTenRankings {
		rankingData := rankingsResponseRankingData{
			OwnerID:  topTenRanking.PID,
			Rank:     i + 1,
			Time:     topTenRanking.Score,
			UserData: topTenRanking.PlayerInfo,
		}

		responseData := rankingsResponseData{
			RankingData: rankingData,
		}

		data = append(data, responseData)
	}

	dataArray := rankingsResponseDataArray{
		NumRecords: numberOfRankings,
		Data:       data,
	}

	writeGetTop10RankingsResponse(raceServiceResultSuccess, responseWriter, dataArray)
}

func handleSubmitScoresRequest(moduleName string, responseWriter http.ResponseWriter, requestBody []byte) {
	requestXML := submitScoresRequestEnvelope{}
	err := xml.Unmarshal(requestBody, &requestXML)
	if err != nil {
		logging.Error(moduleName, "Got malformed XML")
		writeSubmitScoresResponse(raceServiceResultParseError, responseWriter)
		return
	}

	gameData := requestXML.Body.SubmitScores.GameData

	if gameData.GameId != marioKartWiiGameID {
		logging.Error(moduleName, "Wrong GameSpy game ID:", aurora.Cyan(gameData.GameId))
		writeSubmitScoresResponse(raceServiceResultInvalidParameters, responseWriter)
		return
	}

	if gameData.ProfileID <= 0 {
		logging.Error(moduleName, "Invalid profile ID:", aurora.Cyan(gameData.ProfileID))
		writeSubmitScoresResponse(raceServiceResultInvalidParameters, responseWriter)
		return
	}

	if !gameData.RegionId.IsValid() || gameData.RegionId == common.Worldwide {
		logging.Error(moduleName, "Invalid region ID:", aurora.Cyan(gameData.RegionId))
		writeSubmitScoresResponse(raceServiceResultInvalidParameters, responseWriter)
		return
	}

	scoreMode := gameData.ScoreMode
	if scoreMode != scoreModeTimeTrials && scoreMode != scoreModeContest {
		logging.Error(moduleName, "Invalid score mode:", aurora.Cyan(scoreMode))
		writeSubmitScoresResponse(raceServiceResultInvalidParameters, responseWriter)
		return
	}

	for _, scoreData := range gameData.ScoreDatas.ScoreData {
		if scoreData.Time <= 0 || scoreData.Time >= 360000 /* 6 minutes */ {
			logging.Error(moduleName, "Invalid time:", aurora.Cyan(scoreData.Time))
			writeSubmitScoresResponse(raceServiceResultInvalidParameters, responseWriter)
			return
		}

		courseId := scoreData.CourseID
		isContest := scoreMode == scoreModeContest
		if courseId < common.MarioCircuit || isContest == courseId.IsValid() || courseId > 32767 {
			logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseId))
			writeSubmitScoresResponse(raceServiceResultInvalidParameters, responseWriter)
			return
		}

		if !IsPlayerInfoValid(scoreData.PlayerInfoBase64) {
			logging.Error(moduleName, "Invalid player info:", aurora.Cyan(scoreData.PlayerInfoBase64))
			writeSubmitScoresResponse(raceServiceResultInvalidParameters, responseWriter)
			return
		}
	}

	// While we recognize that we have received the time, in contrast to the original Race server,
	// we require that a ghost accompanies a time in order to display it on the rankings.
	writeSubmitScoresResponse(raceServiceResultSuccess, responseWriter)
}

func writeGetTop10RankingsResponse(raceServiceResult raceServiceResult, responseWriter http.ResponseWriter,
	dataArray rankingsResponseDataArray) {
	rankingDataResponse := rankingsResponseRankingDataResponse{
		XMLNSXSI:     xmlNamespaceXSI,
		XMLNSXSD:     xmlNamespaceXSD,
		XMLNS:        xmlNamespace,
		ResponseCode: raceServiceResult,
		DataArray:    dataArray,
	}

	writeResponse(responseWriter, rankingDataResponse)
}

func writeSubmitScoresResponse(raceServiceResult raceServiceResult, responseWriter http.ResponseWriter) {
	submitScoresResponse := submitScoresResponse{
		XMLNSXSI:     xmlNamespaceXSI,
		XMLNSXSD:     xmlNamespaceXSD,
		XMLNS:        xmlNamespace,
		ResponseCode: raceServiceResult,
	}

	writeResponse(responseWriter, submitScoresResponse)
}

func writeResponse(responseWriter http.ResponseWriter, data any) {
	responseBody, err := xml.Marshal(data)
	if err != nil {
		panic(err)
	}

	responseBody = append([]byte(xml.Header), responseBody...)

	responseWriter.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	responseWriter.Header().Set("Content-Type", "text/xml")
	responseWriter.Write(responseBody)
}

func IsPlayerInfoValid(playerInfoString string) bool {
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

	return common.MarioKartWiiControllerId(playerInfo.ControllerId).IsValid()
}
