package race

import (
	"encoding/xml"
	"io"
	"net/http"
	"wwfc/common"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

type rankingsRequestEnvelope struct {
	XMLName xml.Name
	Body    rankingsRequestBody
}

type rankingsRequestBody struct {
	XMLName xml.Name
	Data    rankingsRequestData `xml:",any"`
}

type rankingsRequestData struct {
	XMLName  xml.Name
	GameId   uint                        `xml:"gameid"`
	RegionId common.MarioKartWiiRegionID `xml:"regionid"`
	CourseId common.MarioKartWiiCourseID `xml:"courseid"`
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

	soapMessage := rankingsRequestEnvelope{}
	err = xml.Unmarshal(requestBody, &soapMessage)
	if err != nil {
		logging.Error(moduleName, "Malformed XML")
		return
	}
	soapMessageData := soapMessage.Body.Data

	gameId := soapMessageData.GameId
	if gameId != common.MarioKartWiiGameSpyGameID {
		logging.Error(moduleName, "Wrong GameSpy game id")
		return
	}

	soapAction := soapMessageData.XMLName.Local
	switch soapAction {
	case "GetTopTenRankings":
		regionId := soapMessageData.RegionId
		courseId := soapMessageData.CourseId

		if !regionId.IsValid() {
			logging.Error(moduleName, "Invalid region id")
			return
		}
		if !courseId.IsValid() {
			logging.Error(moduleName, "Invalid course id")
			return
		}

		logging.Info(moduleName, "Received a request for the Top 10 of", aurora.BrightCyan(courseId.ToString()))
	}
}
