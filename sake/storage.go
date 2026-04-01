package sake

import (
	"encoding/xml"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/jackc/pgx/v4"
	"github.com/logrusorgru/aurora/v3"
)

const (
	ResultSuccess             = "Success"
	ResultSecretKeyInvalid    = "SecretKeyInvalid"
	ResultServiceDisabled     = "ServiceDisabled"
	ResultDatabaseUnavailable = "DatabaseUnavailable"
	ResultLoginTicketInvalid  = "LoginTicketInvalid"
	ResultLoginTicketExpired  = "LoginTicketExpired"
	ResultTableNotFound       = "TableNotFound"
	ResultRecordNotFound      = "RecordNotFound"
	ResultFieldNotFound       = "FieldNotFound"
	ResultFieldTypeInvalid    = "FieldTypeInvalid"
	ResultNoPermission        = "NoPermission"
	ResultRecordLimitReached  = "RecordLimitReached"
	ResultAlreadyRated        = "AlreadyRated"
	ResultNotRateable         = "NotRateable"
	ResultNotOwned            = "NotOwned"
	ResultFilterInvalid       = "FilterInvalid"
	ResultSortInvalid         = "SortInvalid"
	ResultTargetFilterInvalid = "TargetFilterInvalid"
	ResultUnknownError        = "UnknownError"
)

const (
	SOAPEnvNamespace = "http://schemas.xmlsoap.org/soap/envelope/"
	SakeNamespace    = "http://gamespy.net/sake"
)

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
	RecordID    int32                     `xml:"recordid"`
	Filter      string                    `xml:"filter"`
	Sort        string                    `xml:"sort"`
	Offset      int                       `xml:"offset"`
	Max         int                       `xml:"max"`
	Surrounding int                       `xml:"surrounding"`
	OwnerIDs    StorageOwnerIDs           `xml:"ownerids"`
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

type StorageOwnerIDs struct {
	OwnerID []int32 `xml:"int"`
}

type StorageResponseEnvelope struct {
	XMLName xml.Name
	Body    StorageResponseBody `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}

type StorageResponseBody struct {
	GetMyRecordsResponse     *StorageGetMyRecordsResponse     `xml:"http://gamespy.net/sake GetMyRecordsResponse"`
	UpdateRecordResponse     *StorageUpdateRecordResponse     `xml:"http://gamespy.net/sake UpdateRecordResponse"`
	SearchForRecordsResponse *StorageSearchForRecordsResponse `xml:"http://gamespy.net/sake SearchForRecordsResponse"`
}

type StorageGetMyRecordsResponse struct {
	GetMyRecordsResult string
	Values             StorageResponseValues `xml:"values"`
}

type StorageResponseValues struct {
	ArrayOfRecordValue []StorageArrayOfRecordValue `xml:"ArrayOfRecordValue"`
}

type StorageArrayOfRecordValue struct {
	RecordValues []StorageRecordValue `xml:"RecordValue"`
}

type StorageUpdateRecordResponse struct {
	UpdateRecordResult string
}

type StorageSearchForRecordsResponse struct {
	XMLName                xml.Name
	SearchForRecordsResult string
	Values                 StorageResponseValues `xml:"values"`
}

var (
	sakeTypeToTag = map[database.SakeFieldType]string{
		database.SakeFieldTypeByte:          "byteValue",
		database.SakeFieldTypeShort:         "shortValue",
		database.SakeFieldTypeInt:           "intValue",
		database.SakeFieldTypeFloat:         "floatValue",
		database.SakeFieldTypeAsciiString:   "asciiStringValue",
		database.SakeFieldTypeUnicodeString: "unicodeStringValue",
		database.SakeFieldTypeBoolean:       "booleanValue",
		database.SakeFieldTypeDateAndTime:   "dateAndTimeValue",
		database.SakeFieldTypeBinaryData:    "binaryDataValue",
		database.SakeFieldTypeInt64:         "int64Value",
	}

	tagToSakeType = common.ReverseMap(sakeTypeToTag).(map[string]database.SakeFieldType)

	storageRequestHandlers = map[string]func(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestData) StorageResponseBody{
		SakeNamespace + "/CreateRecord":     createRecord,
		SakeNamespace + "/UpdateRecord":     updateRecord,
		SakeNamespace + "/GetMyRecords":     getMyRecords,
		SakeNamespace + "/SearchForRecords": searchForRecords,
	}
)

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

	logging.Info("SAKE", "Received storage request with SOAPAction:", aurora.Yellow(headerAction), "and body:", aurora.Cyan(string(body)))

	// Parse the SOAP request XML
	soap := StorageRequestEnvelope{}
	err = xml.Unmarshal(body, &soap)
	if err != nil {
		logging.Error(moduleName, "Received invalid XML")
		return
	}

	response := StorageResponseEnvelope{
		XMLName: xml.Name{Space: SOAPEnvNamespace, Local: "Envelope"},
	}

	xmlName := soap.Body.Data.XMLName.Space + "/" + soap.Body.Data.XMLName.Local
	if headerAction == xmlName || headerAction == `"`+xmlName+`"` {
		logging.Info(moduleName, "SOAPAction:", aurora.Yellow(soap.Body.Data.XMLName.Local))

		handler, ok := storageRequestHandlers[xmlName]
		if !ok {
			panic("unknown SOAPAction: " + aurora.Cyan(xmlName).String())
		}

		profileId, gameInfo, errorString := getRequestIdentity(moduleName, soap.Body.Data)
		if errorString != ResultSuccess {
			logging.Error(moduleName, "Failed to get request identity:", aurora.Cyan(errorString))
			response.Body.setResultTag(xmlName, errorString)
		} else {
			response.Body = handler(moduleName, profileId, gameInfo, soap.Body.Data)
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

func getRequestIdentity(moduleName string, request StorageRequestData) (uint32, common.GameInfo, string) {
	gameInfo := common.GetGameInfoByID(request.GameID)
	if gameInfo == nil {
		logging.Error(moduleName, "Invalid game ID:", aurora.Cyan(request.GameID))
		return 0, common.GameInfo{}, ResultDatabaseUnavailable
	}

	if gameInfo.SecretKey != request.SecretKey {
		logging.Error(moduleName, "Mismatch", aurora.BrightCyan(gameInfo.Name), "secret key:", aurora.Cyan(request.SecretKey), "!=", aurora.BrightCyan(gameInfo.SecretKey))
		return 0, common.GameInfo{}, ResultSecretKeyInvalid
	}

	profileId, issueTime, err := common.UnmarshalGPCMLoginTicket(request.LoginTicket)
	if err != nil {
		logging.Error(moduleName, err)
		return 0, common.GameInfo{}, ResultLoginTicketInvalid
	}

	if issueTime.Add(48 * time.Hour).Before(time.Now()) {
		return 0, common.GameInfo{}, ResultLoginTicketExpired
	}

	return profileId, *gameInfo, ResultSuccess
}

func getInputFields(moduleName string, request StorageRequestData) (map[string]database.SakeField, string) {
	record := database.SakeRecord{
		Fields: make(map[string]database.SakeField),
	}
	for _, field := range request.Values.RecordFields {
		fieldType, ok := tagToSakeType[field.Value.Value.XMLName.Local]
		if !ok {
			logging.Error(moduleName, "Invalid field type tag:", aurora.Cyan(field.Value.Value.XMLName.Local))
			return nil, ResultFieldTypeInvalid
		}
		if field.Name == "ownerid" || field.Name == "recordid" {
			logging.Error(moduleName, "Attempt to set reserved field:", aurora.Cyan(field.Name))
			return nil, ResultNoPermission
		}
		record.Fields[field.Name] = database.SakeField{Type: fieldType, Value: field.Value.Value.Value}
	}
	return record.Fields, ResultSuccess
}

func createRecord(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestData) StorageResponseBody {
	if request.TableID == "" {
		logging.Error(moduleName, "No table ID provided")
		return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
			UpdateRecordResult: ResultTableNotFound,
		}}
	}

	if gameInfo.Name == "mariokartwii" && (request.TableID == "GhostData" || request.TableID == "StoredGhostData") {
		// Reserved for special handler
		return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
			UpdateRecordResult: ResultTableNotFound,
		}}
	}

	var record database.SakeRecord
	var result string
	record.Fields, result = getInputFields(moduleName, request)
	if result != ResultSuccess {
		return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
			UpdateRecordResult: result,
		}}
	}

	record.GameId = gameInfo.GameID
	record.TableId = request.TableID
	record.RecordId = 0
	record.OwnerId = int32(profileId)

	// TODO: Limit number of records or fields a user can have
	err := database.InsertSakeRecord(pool, ctx, record)
	if err != nil {
		return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
			UpdateRecordResult: ResultDatabaseUnavailable,
		}}
	}

	return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
		UpdateRecordResult: ResultSuccess,
	}}

}

func getMyRecords(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestData) StorageResponseBody {
	if len(request.Fields.Fields) == 0 {
		// GameSpy client doesn't consider zero fields valid
		return StorageResponseBody{GetMyRecordsResponse: &StorageGetMyRecordsResponse{
			GetMyRecordsResult: "BadNumFields",
		}}
	}
	if request.TableID == "" {
		logging.Error(moduleName, "No table ID provided")
		return StorageResponseBody{GetMyRecordsResponse: &StorageGetMyRecordsResponse{
			GetMyRecordsResult: ResultTableNotFound,
		}}
	}

	records, err := database.GetSakeRecords(pool, ctx, gameInfo.GameID, []int32{int32(profileId)}, request.TableID, nil, request.Fields.Fields, request.Filter)
	if err != nil {
		logging.Error(moduleName, "Failed to get sake records from the database:", err)
		if err == pgx.ErrNoRows {
			return StorageResponseBody{GetMyRecordsResponse: &StorageGetMyRecordsResponse{
				GetMyRecordsResult: ResultRecordNotFound,
			}}
		}
		return StorageResponseBody{GetMyRecordsResponse: &StorageGetMyRecordsResponse{
			GetMyRecordsResult: ResultDatabaseUnavailable,
		}}
	}

	response := StorageGetMyRecordsResponse{
		GetMyRecordsResult: ResultSuccess,
		Values:             fillResponseValues(records, request),
	}
	logging.Info(moduleName, "Returning", aurora.Cyan(len(records)), "records from table", aurora.Cyan(request.TableID), "for profile", aurora.Cyan(profileId))

	return StorageResponseBody{GetMyRecordsResponse: &response}
}

func updateRecord(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestData) StorageResponseBody {
	if request.TableID == "" {
		logging.Error(moduleName, "No table ID provided")
		return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
			UpdateRecordResult: ResultTableNotFound,
		}}
	}

	if gameInfo.Name == "mariokartwii" && (request.TableID == "GhostData" || request.TableID == "StoredGhostData") {
		// Reserved for special handler
		return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
			UpdateRecordResult: ResultTableNotFound,
		}}
	}

	var record database.SakeRecord
	var result string
	record.Fields, result = getInputFields(moduleName, request)
	if result != ResultSuccess {
		return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
			UpdateRecordResult: result,
		}}
	}

	record.GameId = gameInfo.GameID
	record.TableId = request.TableID
	record.RecordId = request.RecordID
	record.OwnerId = int32(profileId)

	err := database.UpdateSakeRecord(pool, ctx, record, int32(profileId))
	if err != nil {
		logging.Error(moduleName, "Failed to update sake record in the database:", err)
		if err == database.ErrSakeNotOwned {
			return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
				UpdateRecordResult: ResultNotOwned,
			}}
		}
		if err == pgx.ErrNoRows {
			return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
				UpdateRecordResult: ResultRecordNotFound,
			}}
		}
		return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
			UpdateRecordResult: ResultDatabaseUnavailable,
		}}
	}

	logging.Info(moduleName, "Updated record", aurora.Cyan(record.RecordId), "in table", aurora.Cyan(record.TableId), "for profile", aurora.Cyan(profileId))

	return StorageResponseBody{UpdateRecordResponse: &StorageUpdateRecordResponse{
		UpdateRecordResult: ResultSuccess,
	}}
}

var searchRecordHandlers = map[string]func(string, StorageRequestData) (database.SakeRecord, bool){
	"mariokartwii/GhostData":       getMarioKartWiiGhostDataRecord,
	"mariokartwii/StoredGhostData": getMarioKartWiiStoredGhostDataRecord,
}

var (
	standardFilterRegex = regexp.MustCompile(`^ownerid\s*=\s*-?(\d{1,10})$`)
)

// TODO: Test this
func searchForRecords(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestData) StorageResponseBody {
	var records []database.SakeRecord

	if handler, ok := searchRecordHandlers[gameInfo.Name+"/"+request.TableID]; ok {
		record, ok := handler(moduleName, request)
		if !ok {
			return StorageResponseBody{SearchForRecordsResponse: &StorageSearchForRecordsResponse{
				SearchForRecordsResult: ResultUnknownError,
			}}
		}
		records = append(records, record)
	} else {
		filter := request.Filter
		ownerIds := request.OwnerIDs.OwnerID

		// Parse standard filter
		if matches := standardFilterRegex.FindStringSubmatch(request.Filter); matches != nil {
			ownerIdParsed, err := strconv.ParseInt(matches[1], 10, 32)
			if err != nil {
				logging.Error(moduleName, "Invalid owner ID in filter:", aurora.Cyan(request.Filter))
				return StorageResponseBody{SearchForRecordsResponse: &StorageSearchForRecordsResponse{
					SearchForRecordsResult: ResultFilterInvalid,
				}}
			}
			ownerIds = append(ownerIds, int32(ownerIdParsed))
			filter = ""
		}

		var err error
		records, err = database.GetSakeRecords(pool, ctx, gameInfo.GameID, ownerIds, request.TableID, nil, request.Fields.Fields, filter)
		if err != nil {
			logging.Error(moduleName, "Failed to get sake records from the database:", err)
			return StorageResponseBody{SearchForRecordsResponse: &StorageSearchForRecordsResponse{
				SearchForRecordsResult: ResultDatabaseUnavailable,
			}}
		}
	}

	// Sort the records now
	sort.Slice(records, func(l, r int) bool {
		lVal, lExists := records[l].Fields[request.Sort]
		rVal, rExists := records[r].Fields[request.Sort]
		if !lExists || !rExists {
			// Prioritises the one that exists or goes left if both false
			return rExists
		}

		if lVal.Type != database.SakeFieldTypeInt || rVal.Type != database.SakeFieldTypeInt {
			panic(aurora.Cyan(lVal.Type).String() + " used as sort value")
		}

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

	// Enforce the maximum number of records after sorting
	if request.Max > 0 && len(records) > request.Max {
		records = records[:request.Max]
	}

	response := StorageSearchForRecordsResponse{
		SearchForRecordsResult: "Success",
		Values:                 fillResponseValues(records, request),
	}
	logging.Info(moduleName, "Searched for records in table", aurora.Cyan(request.TableID), "for profile", aurora.Cyan(profileId), "with filter", aurora.Cyan(request.Filter), "and got", aurora.Cyan(len(records)), "records")

	return StorageResponseBody{SearchForRecordsResponse: &response}
}

func fillResponseValues(records []database.SakeRecord, request StorageRequestData) StorageResponseValues {
	var response StorageResponseValues
	for _, record := range records {
		valueArray := StorageArrayOfRecordValue{}
		for _, field := range request.Fields.Fields {
			if field == "ownerid" {
				valueArray.RecordValues = append(valueArray.RecordValues, StorageRecordValue{Value: &StorageValue{
					XMLName: xml.Name{Local: "intValue"},
					Value:   strconv.FormatInt(int64(int32(record.OwnerId)), 10),
				}})
				continue
			}
			if field == "recordid" {
				valueArray.RecordValues = append(valueArray.RecordValues, StorageRecordValue{Value: &StorageValue{
					XMLName: xml.Name{Local: "intValue"},
					Value:   strconv.FormatInt(int64(int32(record.RecordId)), 10),
				}})
				continue
			}

			var fieldValue *database.SakeField
			for name, value := range record.Fields {
				if name == field {
					fieldValue = &value
					break
				}
			}
			if fieldValue == nil {
				valueArray.RecordValues = append(valueArray.RecordValues, StorageRecordValue{Value: nil})
			} else {
				value := fillValue(fieldValue.Type, fieldValue.Value)
				valueArray.RecordValues = append(valueArray.RecordValues, StorageRecordValue{Value: &value})
			}
		}
		response.ArrayOfRecordValue = append(response.ArrayOfRecordValue, valueArray)
	}
	return response
}

func fillValue(valueType database.SakeFieldType, value string) StorageValue {
	return StorageValue{
		XMLName: xml.Name{Local: sakeTypeToTag[valueType]},
		Value:   value,
	}
}

func (body *StorageResponseBody) setResultTag(xmlName string, result string) {
	switch xmlName {
	case SakeNamespace + "/GetMyRecords":
		body.GetMyRecordsResponse = &StorageGetMyRecordsResponse{
			GetMyRecordsResult: result,
		}

	case SakeNamespace + "/UpdateRecord":
		body.UpdateRecordResponse = &StorageUpdateRecordResponse{
			UpdateRecordResult: result,
		}

	case SakeNamespace + "/SearchForRecords":
		body.SearchForRecordsResponse = &StorageSearchForRecordsResponse{}
	}
}
