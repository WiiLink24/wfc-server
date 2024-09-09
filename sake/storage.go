package sake

import (
	"encoding/base64"
	"encoding/xml"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

const (
	FileRequestDownload = iota
	FileRequestUpload
)

const (
	SOAPEnvNamespace = "http://schemas.xmlsoap.org/soap/envelope/"
	SakeNamespace    = "http://gamespy.net/sake"
)

type FileRequest int

type StorageRequestEnvelope struct {
	XMLName xml.Name
	Body    StorageRequestBody
}

type StorageRequestBody struct {
	XMLName xml.Name
	Data    StorageRequestData `xml:",any"`
}

type StorageRequestData struct {
	XMLName     xml.Name
	GameID      int                       `xml:"gameid"`
	SecretKey   string                    `xml:"secretKey"`
	LoginTicket string                    `xml:"loginTicket"`
	TableID     string                    `xml:"tableid"`
	RecordID    string                    `xml:"recordid"`
	Filter      string                    `xml:"filter"`
	Sort        string                    `xml:"sort"`
	Offset      int                       `xml:"offset"`
	Max         int                       `xml:"max"`
	Surrounding int                       `xml:"surrounding"`
	OwnerIDs    string                    `xml:"ownerids"`
	CacheFlag   int                       `xml:"cacheFlag"`
	Fields      StorageFields             `xml:"fields"`
	Values      StorageUpdateRecordValues `xml:"values"`
}

type StorageFields struct {
	XMLName xml.Name
	Fields  []string `xml:"string"`
}

type StorageUpdateRecordValues struct {
	RecordFields []StorageRecordField `xml:"RecordField"`
}

type StorageRecordField struct {
	Name  string             `xml:"name"`
	Value StorageRecordValue `xml:"value"`
}

type StorageRecordValue struct {
	XMLName xml.Name
	Value   *StorageValue `xml:",any"`
}

type StorageValue struct {
	XMLName xml.Name
	Value   string `xml:"value"`
}

type StorageResponseEnvelope struct {
	XMLName xml.Name
	Body    StorageResponseBody
}

type StorageResponseBody struct {
	XMLName                  xml.Name
	GetMyRecordsResponse     *StorageGetMyRecordsResponse     `xml:"http://gamespy.net/sake GetMyRecordsResponse"`
	UpdateRecordResponse     *StorageUpdateRecordResponse     `xml:"http://gamespy.net/sake UpdateRecordResponse"`
	SearchForRecordsResponse *StorageSearchForRecordsResponse `xml:"http://gamespy.net/sake SearchForRecordsResponse"`
}

type StorageGetMyRecordsResponse struct {
	XMLName            xml.Name
	GetMyRecordsResult string
	Values             StorageResponseValues `xml:"values"` // ???
}

type StorageResponseValues struct {
	XMLName            xml.Name
	ArrayOfRecordValue StorageArrayOfRecordValue
}

type StorageArrayOfRecordValue struct {
	XMLName      xml.Name
	RecordValues []StorageRecordValue `xml:"RecordValue"`
}

type StorageUpdateRecordResponse struct {
	XMLName            xml.Name
	UpdateRecordResult string
	// TODO
}

type StorageSearchForRecordsResponse struct {
	XMLName                xml.Name
	SearchForRecordsResult string
	Values                 StorageResponseValues `xml:"values"` // ???
}

var fileDownloadHandlers = map[int]func(string, http.ResponseWriter, *http.Request){
	common.GetGameIDOrPanic("mariokartwii"): handleMarioKartWiiFileDownloadRequest,
}

var fileUploadHandlers = map[int]func(string, http.ResponseWriter, *http.Request){
	common.GetGameIDOrPanic("mariokartwii"): handleMarioKartWiiFileUploadRequest,
}

func handleStorageRequest(moduleName string, w http.ResponseWriter, r *http.Request) {
	headerAction := r.Header.Get("SOAPAction")
	if headerAction == "" {
		logging.Error(moduleName, "No SOAPAction in header")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	// Parse the SOAP request XML
	soap := StorageRequestEnvelope{}
	err = xml.Unmarshal(body, &soap)
	if err != nil {
		logging.Error(moduleName, "Received invalid XML")
		return
	}

	response := StorageResponseEnvelope{
		XMLName: xml.Name{Space: SOAPEnvNamespace, Local: "Envelope"},
		Body: StorageResponseBody{
			XMLName: xml.Name{Space: SOAPEnvNamespace, Local: "Body"},
		},
	}

	xmlName := soap.Body.Data.XMLName.Space + "/" + soap.Body.Data.XMLName.Local
	if headerAction == xmlName || headerAction == `"`+xmlName+`"` {
		logging.Info(moduleName, "SOAPAction:", aurora.Yellow(soap.Body.Data.XMLName.Local))

		if profileId, gameInfo, ok := getRequestIdentity(moduleName, soap.Body.Data); ok {
			switch xmlName {
			case SakeNamespace + "/GetMyRecords":
				response.Body.GetMyRecordsResponse = getMyRecords(moduleName, profileId, gameInfo, soap.Body.Data)

			case SakeNamespace + "/UpdateRecord":
				response.Body.UpdateRecordResponse = updateRecord(moduleName, profileId, gameInfo, soap.Body.Data)

			case SakeNamespace + "/SearchForRecords":
				response.Body.SearchForRecordsResponse = searchForRecords(moduleName, gameInfo, soap.Body.Data)

			default:
				logging.Error(moduleName, "Unknown SOAPAction:", aurora.Cyan(xmlName))
			}
		}
	} else {
		logging.Error(moduleName, "Invalid SOAPAction or XML request:", aurora.Cyan(headerAction))
	}

	out, err := xml.Marshal(response)
	if err != nil {
		panic(err)
	}

	payload := append([]byte(xml.Header), out...)

	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Write(payload)
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

func getRequestIdentity(moduleName string, request StorageRequestData) (uint32, common.GameInfo, bool) {
	gameInfo := common.GetGameInfoByID(request.GameID)
	if gameInfo == nil {
		logging.Error(moduleName, "Invalid game ID:", aurora.Cyan(request.GameID))
		return 0, common.GameInfo{}, false
	}

	if gameInfo.SecretKey != request.SecretKey {
		logging.Error(moduleName, "Mismatch", aurora.BrightCyan(gameInfo.Name), "secret key:", aurora.Cyan(request.SecretKey), "!=", aurora.BrightCyan(gameInfo.SecretKey))
		return 0, common.GameInfo{}, false
	}

	profileId, _, err := common.UnmarshalGPCMLoginTicket(request.LoginTicket)
	if err != nil {
		logging.Error(moduleName, err)
		return 0, common.GameInfo{}, false
	}

	logging.Info(moduleName, "Profile ID:", aurora.BrightCyan(profileId))
	logging.Info(moduleName, "Game:", aurora.Cyan(request.GameID), "-", aurora.BrightCyan(gameInfo.Name))
	logging.Info(moduleName, "Table ID:", aurora.Cyan(request.TableID))

	return profileId, *gameInfo, true
}

func binaryDataValue(value []byte) StorageValue {
	return StorageValue{
		XMLName: xml.Name{Local: "binaryDataValue"},
		Value:   base64.StdEncoding.EncodeToString(value),
	}
}

func binaryDataValueBase64(value string) StorageValue {
	return StorageValue{
		XMLName: xml.Name{Local: "binaryDataValue"},
		Value:   value,
	}
}

func intValue(value int32) StorageValue {
	return StorageValue{
		XMLName: xml.Name{Local: "intValue"},
		Value:   strconv.FormatInt(int64(value), 10),
	}
}

// I don't even know if this is a thing
func uintValue(value uint32) StorageValue {
	return StorageValue{
		XMLName: xml.Name{Local: "uintValue"},
		Value:   strconv.FormatUint(uint64(value), 10),
	}
}

func getMyRecords(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestData) *StorageGetMyRecordsResponse {
	errorResponse := StorageGetMyRecordsResponse{
		GetMyRecordsResult: "Error",
	}

	var values map[string]StorageValue

	switch gameInfo.Name + "/" + request.TableID {
	default:
		logging.Error(moduleName, "Unknown table")
		for _, field := range request.Fields.Fields {
			logging.Info(moduleName, "Field:", aurora.Cyan(field))
		}
		return &errorResponse

	case "mariokartwii/FriendInfo":
		// Mario Kart Wii friend info
		values = map[string]StorageValue{
			"ownerid":  uintValue(profileId),
			"recordid": intValue(int32(profileId)),
			"info":     binaryDataValueBase64(database.GetMKWFriendInfo(pool, ctx, profileId)),
		}
	}

	response := StorageGetMyRecordsResponse{
		GetMyRecordsResult: "Success",
	}

	fieldCount := 0
	valueArray := &response.Values.ArrayOfRecordValue
	for _, field := range request.Fields.Fields {
		if value, ok := values[field]; ok {
			fieldCount++
			valueArray.RecordValues = append(valueArray.RecordValues, StorageRecordValue{Value: &value})
		} else {
			valueArray.RecordValues = append(valueArray.RecordValues, StorageRecordValue{Value: nil})
		}
	}

	logging.Info(moduleName, "Wrote", aurora.Cyan(fieldCount), "field(s)")
	return &response
}

func updateRecord(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestData) *StorageUpdateRecordResponse {
	errorResponse := StorageUpdateRecordResponse{
		UpdateRecordResult: "Error",
	}

	switch gameInfo.Name + "/" + request.TableID {
	default:
		logging.Error(moduleName, "Unknown table")
		for _, field := range request.Values.RecordFields {
			logging.Info(moduleName, "Field:", aurora.Cyan(field.Name), "Type:", aurora.Cyan(field.Value.XMLName.Local), "Value:", aurora.Cyan(field.Value.Value.Value))
		}
		return &errorResponse

	case "mariokartwii/FriendInfo":
		// Mario Kart Wii friend info
		if len(request.Values.RecordFields) != 1 || request.Values.RecordFields[0].Name != "info" || request.Values.RecordFields[0].Value.Value.XMLName.Local != "binaryDataValue" {
			logging.Error(moduleName, "Invalid record fields")
			return &errorResponse
		}

		// TODO: Validate record data
		database.UpdateMKWFriendInfo(pool, ctx, profileId, request.Values.RecordFields[0].Value.Value.Value)
		logging.Notice(moduleName, "Updated Mario Kart Wii friend info")
	}

	return &StorageUpdateRecordResponse{
		UpdateRecordResult: "Success",
	}
}

func searchForRecords(moduleName string, gameInfo common.GameInfo, request StorageRequestData) *StorageSearchForRecordsResponse {
	errorResponse := StorageSearchForRecordsResponse{
		SearchForRecordsResult: "Error",
	}

	var values []map[string]StorageValue

	switch gameInfo.Name + "/" + request.TableID {
	default:
		logging.Error(moduleName, "Unknown table")
		for _, field := range request.Fields.Fields {
			logging.Info(moduleName, "Field:", aurora.Cyan(field))
		}
		return &errorResponse

	case "mariokartwii/FriendInfo":
		// Mario Kart Wii friend info
		match := regexp.MustCompile(`^ownerid = (\d{1,10})$`).FindStringSubmatch(request.Filter)
		if len(match) != 2 {
			logging.Error(moduleName, "Invalid filter")
			return &errorResponse
		}

		ownerId, err := strconv.ParseInt(match[1], 10, 32)
		if err != nil {
			logging.Error(moduleName, "Invalid owner ID")
			return &errorResponse
		}

		values = []map[string]StorageValue{
			{
				"ownerid":  uintValue(uint32(ownerId)),
				"recordid": intValue(int32(ownerId)),
				"info":     binaryDataValueBase64(database.GetMKWFriendInfo(pool, ctx, uint32(ownerId))),
			},
		}

	case "mariokartwii/StoredGhostData":
		if request.Sort != "time" {
			logging.Error(moduleName, "Invalid sort string:", aurora.Cyan(request.Sort))
			return &errorResponse
		}

		if request.Offset != 0 {
			logging.Error(moduleName, "Invalid offset value:", aurora.Cyan(request.Offset))
			return &errorResponse
		}

		if request.Max != 1 {
			logging.Error(moduleName, "Invalid number of records to return:", aurora.Cyan(request.Max))
			return &errorResponse
		}

		if request.Surrounding != 0 {
			logging.Error(moduleName, "Invalid number of surrounding records to return:", aurora.Cyan(request.Surrounding))
			return &errorResponse
		}

		if request.OwnerIDs != "" {
			logging.Error(moduleName, "Invalid owner id array:", aurora.Cyan(request.OwnerIDs))
			return &errorResponse
		}

		if request.CacheFlag != 0 {
			logging.Error(moduleName, "Invalid cache value:", aurora.Cyan(request.CacheFlag))
			return &errorResponse
		}

		match := regexp.MustCompile(`^course = ([1-9]\d?|0) and gameid = 1687(?: and region = ([1-7]))?$`).FindStringSubmatch(request.Filter)
		if match == nil {
			logging.Error(moduleName, "Invalid filter string:", aurora.Cyan(request.Filter))
			return &errorResponse
		}

		courseIdInt, _ := strconv.Atoi(match[1])
		courseId := common.MarioKartWiiCourseId(courseIdInt)
		if !courseId.IsValid() {
			logging.Error(moduleName, "Invalid course ID:", aurora.Cyan(courseIdInt))
			return &errorResponse
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
			logging.Error(moduleName, "Failed to get the stored ghost data from the database:", err)
			return &errorResponse
		}

		values = []map[string]StorageValue{
			{
				"profile": intValue(int32(pid)),
				"fileid":  intValue(int32(fileId)),
			},
		}
	}

	// Sort the values now
	sort.Slice(values, func(l, r int) bool {
		lVal, lExists := values[l][request.Sort]
		rVal, rExists := values[r][request.Sort]
		if !lExists || !rExists {
			// Prioritises the one that exists or goes left if both false
			return rExists
		}

		if lVal.XMLName.Local != "intValue" && lVal.XMLName.Local != "uintValue" {
			panic(aurora.Cyan(lVal.XMLName.Local).String() + " used as sort value")
		}
		// Assuming the two use the same type

		lValInt, err := strconv.ParseInt(lVal.Value, 10, 64)
		if err != nil {
			panic(err)
		}
		rValInt, err := strconv.ParseInt(rVal.Value, 10, 64)
		if err != nil {
			panic(err)
		}

		return lValInt < rValInt
	})

	response := StorageSearchForRecordsResponse{
		SearchForRecordsResult: "Success",
	}

	fieldCount := 0
	valueArray := &response.Values.ArrayOfRecordValue
	var i int
	for i = 0; i < len(values) && i < request.Max; i++ {
		for _, field := range request.Fields.Fields {
			if value, ok := values[i][field]; ok {
				fieldCount++
				valueArray.RecordValues = append(valueArray.RecordValues, StorageRecordValue{Value: &value})
			} else {
				valueArray.RecordValues = append(valueArray.RecordValues, StorageRecordValue{Value: nil})
			}
		}
	}

	logging.Info(moduleName, "Wrote", aurora.BrightCyan(fieldCount), "field(s) across", aurora.BrightCyan(i), "record(s)")
	return &response
}
