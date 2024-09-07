package race

import (
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
	Body rankingsRequestBody
}

type rankingsRequestBody struct {
	Data rankingsRequestData `xml:",any"`
}

type rankingsRequestData struct {
	XMLName  xml.Name
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

type raceServiceResult int

// https://github.com/GameProgressive/UniSpySDK/blob/master/webservices/RacingService.h
const (
	raceServiceResultSuccess           = 0
	raceServiceResultDatabaseError     = 6
	raceServiceResultParseError        = 101
	raceServiceResultInvalidParameters = 105
)

const (
	xmlNamespaceXSI = "http://www.w3.org/2001/XMLSchema-instance"
	xmlNamespaceXSD = "http://www.w3.org/2001/XMLSchema"
	xmlNamespace    = "http://gamespy.net/RaceService/"
)

var MarioKartWiiGameID = common.GetGameIDOrPanic("mariokartwii") // 1687

func handleNintendoRacingServiceRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	soapActionHeader := request.Header.Get("SOAPAction")
	if soapActionHeader == "" {
		logging.Error(moduleName, "No SOAPAction header")
		writeErrorResponse(raceServiceResultParseError, responseWriter)
		return
	}

	slashIndex := strings.LastIndex(soapActionHeader, "/")
	if slashIndex == -1 {
		logging.Error(moduleName, "Invalid SOAPAction header")
		writeErrorResponse(raceServiceResultParseError, responseWriter)
		return
	}
	quotationMarkIndex := strings.Index(soapActionHeader[slashIndex+1:], "\"")
	if quotationMarkIndex == -1 {
		logging.Error(moduleName, "Invalid SOAPAction header")
		writeErrorResponse(raceServiceResultParseError, responseWriter)
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

	// TODO SubmitScores
	default:
		logging.Info(moduleName, "Unhandled SOAPAction:", aurora.Cyan(soapAction))
	}
}

func handleGetTopTenRankingsRequest(moduleName string, responseWriter http.ResponseWriter, requestBody []byte) {
	requestXML := rankingsRequestEnvelope{}
	err := xml.Unmarshal(requestBody, &requestXML)
	if err != nil {
		logging.Error(moduleName, "Got malformed XML")
		writeErrorResponse(raceServiceResultParseError, responseWriter)
		return
	}

	requestData := requestXML.Body.Data

	gameId := requestData.GameId
	if gameId != MarioKartWiiGameID {
		logging.Error(moduleName, "Wrong GameSpy game ID:", aurora.Cyan(gameId))
		writeErrorResponse(raceServiceResultInvalidParameters, responseWriter)
		return
	}

	regionId := requestData.RegionId
	courseId := requestData.CourseId

	if !regionId.IsValid() {
		logging.Error(moduleName, "Invalid region ID:", aurora.Cyan(regionId))
		writeErrorResponse(raceServiceResultInvalidParameters, responseWriter)
		return
	}
	if courseId < common.MarioCircuit || courseId > 32767 {
		logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseId))
		writeErrorResponse(raceServiceResultInvalidParameters, responseWriter)
		return
	}

	topTenRankings, err := database.GetMarioKartWiiTopTenRankings(pool, ctx, regionId, courseId)
	if err != nil {
		logging.Error(moduleName, "Failed to get the Top 10 rankings:", err)
		writeErrorResponse(raceServiceResultDatabaseError, responseWriter)
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

	rankingDataResponse := rankingsResponseRankingDataResponse{
		XMLNSXSI:     xmlNamespaceXSI,
		XMLNSXSD:     xmlNamespaceXSD,
		XMLNS:        xmlNamespace,
		ResponseCode: raceServiceResultSuccess,
		DataArray:    dataArray,
	}

	writeResponse(responseWriter, rankingDataResponse)
}

func writeErrorResponse(raceServiceResult raceServiceResult, responseWriter http.ResponseWriter) {
	rankingDataResponse := rankingsResponseRankingDataResponse{
		XMLNSXSI:     xmlNamespaceXSI,
		XMLNSXSD:     xmlNamespaceXSD,
		XMLNS:        xmlNamespace,
		ResponseCode: raceServiceResult,
	}

	writeResponse(responseWriter, rankingDataResponse)
}

func writeResponse(responseWriter http.ResponseWriter, rankingDataResponse rankingsResponseRankingDataResponse) {
	responseBody, err := xml.Marshal(rankingDataResponse)
	if err != nil {
		panic(err)
	}

	responseBody = append([]byte(xml.Header), responseBody...)

	responseWriter.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	responseWriter.Header().Set("Content-Type", "text/xml")
	responseWriter.Write(responseBody)
}
