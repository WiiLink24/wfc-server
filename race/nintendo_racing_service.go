package race

import (
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"wwfc/common"
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
	GameId   int                         `xml:"gameid"`
	RegionId common.MarioKartWiiRegionID `xml:"regionid"`
	CourseId common.MarioKartWiiCourseID `xml:"courseid"`
}

type rankingsResponseRankingDataResponse struct {
	XMLName      xml.Name `xml:"RankingDataResponse"`
	XMLNSXsi     string   `xml:"xmlns:xsi,attr"`
	XMLNSXsd     string   `xml:"xmlns:xsd,attr"`
	XMLNS        string   `xml:"xmlns,attr"`
	ResponseCode int      `xml:"responseCode"`
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

func handleNintendoRacingServiceRequest(moduleName string, responseWriter http.ResponseWriter, request *http.Request) {
	soapActionHeader := request.Header.Get("SOAPAction")
	if soapActionHeader == "" {
		logging.Error(moduleName, "No SOAPAction header")
		return
	}

	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		panic(err)
	}

	requestXML := rankingsRequestEnvelope{}
	err = xml.Unmarshal(requestBody, &requestXML)
	if err != nil {
		logging.Error(moduleName, "Got malformed XML")
		return
	}
	requestData := requestXML.Body.Data

	gameId := requestData.GameId
	if gameId != common.MarioKartWiiGameSpyGameID {
		logging.Error(moduleName, "Wrong GameSpy game id")
		return
	}

	soapAction := requestData.XMLName.Local
	switch soapAction {
	case "GetTopTenRankings":
		regionId := requestData.RegionId
		courseId := requestData.CourseId

		if !regionId.IsValid() {
			logging.Error(moduleName, "Invalid region id")
			return
		}
		if courseId < common.MarioCircuit {
			logging.Error(moduleName, "Invalid course id")
			return
		}

		var topTenLeaderboard string
		if courseId <= common.GBAShyGuyBeach {
			topTenLeaderboard = courseId.ToString()
		} else {
			topTenLeaderboard = "a competition"
		}

		logging.Info(moduleName, "Received a request for the Top 10 of", aurora.BrightCyan(topTenLeaderboard))
		handleGetTopTenRankingsRequest(moduleName, responseWriter)
	}
}

func handleGetTopTenRankingsRequest(moduleName string, responseWriter http.ResponseWriter) {
	rankingData := rankingsResponseRankingData{
		OwnerID:  1000000404,
		Rank:     1,
		Time:     0,
		UserData: "xC0AIABNAGkAawBlAFMAdABhAHIAIAAAhH/RTQAAAAAgB45hkAAQTEDyjqQAeLgPhq4AiiUEACAATQBpAGsAZQBTAHQAYQByACD0UwACAAE=",
	}

	responseData := []rankingsResponseData{
		{
			RankingData: rankingData,
		},
	}

	dataArray := rankingsResponseDataArray{
		NumRecords: len(responseData),
		Data:       responseData,
	}

	rankingDataResponse := rankingsResponseRankingDataResponse{
		XMLNSXsi:     "http://www.w3.org/2001/XMLSchema-instance",
		XMLNSXsd:     "http://www.w3.org/2001/XMLSchema",
		XMLNS:        "http://gamespy.net/RaceService/",
		ResponseCode: 0,
		DataArray:    dataArray,
	}

	responseBody, err := xml.Marshal(rankingDataResponse)
	if err != nil {
		logging.Error(moduleName, "Failed to XML encode the data")
		return
	}

	responseBody = append([]byte(xml.Header), responseBody...)

	responseWriter.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	responseWriter.Header().Set("Content-Type", "text/xml")
	responseWriter.Write(responseBody)
}
