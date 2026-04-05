package sake

import (
	"encoding/xml"
	"io"
	"net/http"
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
	SOAPEnvNamespace = "http://schemas.xmlsoap.org/soap/envelope/"
	SakeNamespace    = "http://gamespy.net/sake"
)

const (
	ResultSuccess             = Result("Success")             // 4xx51
	ResultDatabaseUnavailable = Result("DatabaseUnavailable") // 4xx58
	ResultLoginTicketInvalid  = Result("LoginTicketInvalid")  // 4xx59
	ResultLoginTicketExpired  = Result("LoginTicketExpired")  // 4xx60
	ResultSecretKeyInvalid    = Result("SecretKeyInvalid")    // 4xx52
	ResultTableNotFound       = Result("TableNotFound")       // 4xx61
	ResultRecordNotFound      = Result("RecordNotFound")      // 4xx62
	ResultFieldNotFound       = Result("FieldNotFound")       // 4xx63
	ResultFieldTypeInvalid    = Result("FieldTypeInvalid")    // 4xx64
	ResultNoPermission        = Result("NoPermission")        // 4xx65
	ResultRecordLimitReached  = Result("RecordLimitReached")  // 4xx66
	ResultAlreadyRated        = Result("AlreadyRated")        // 4xx67
	ResultNotRateable         = Result("NotRateable")         // 4xx68
	ResultNotOwned            = Result("NotOwned")            // 4xx69
	ResultFilterInvalid       = Result("FilterInvalid")       // 4xx70
	ResultSortInvalid         = Result("SortInvalid")         // 4xx71
	ResultTargetFilterInvalid = Result("TargetFilterInvalid") // 4xx80
	ResultUnknownError        = Result("UnknownError")        // 4xx72
	ResultAlreadyReported     = Result("AlreadyReported")     // 4xx72
	ResultNotModerated        = Result("NotModerated")        // 4xx72
	ResultCategoryInvalid     = Result("CategoryInvalid")     // 4xx72
	ResultDuplicateRecord     = Result("DuplicateRecord")     // 4xx72
	ResultServiceDisabled     = Result("ServiceDisabled")     // 4xx53
)

type Result string

type StorageRequestEnvelope struct {
	XMLName xml.Name           `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    StorageRequestBody `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}

type StorageRequestBody struct {
	Data StorageRequestCommon `xml:",any"`
}

// Combined struct for all of the different operations:
// CreateRecord, UpdateRecord, DeleteRecord, SearchForRecords, GetMyRecords
// GetSpecificRecords, GetRandomRecords, GetRecordCount, RateRecord, GetRecordLimit
type StorageRequestCommon struct {
	XMLName      xml.Name
	GameID       int32         `xml:"gameid"`
	LoginTicket  string        `xml:"loginTicket"`
	SecretKey    string        `xml:"secretKey"`
	TableID      string        `xml:"tableid"`
	RecordID     int32         `xml:"recordid"`
	Fields       ArrayOfString `xml:"fields"`
	Filter       string        `xml:"filter"`
	Sort         string        `xml:"sort"`
	Offset       int32         `xml:"offset"`
	Max          int32         `xml:"max"`
	TargetFilter string        `xml:"targetFilter"`
	Surrounding  int32         `xml:"surrounding"`
	OwnerIDs     ArrayOfInt    `xml:"ownerids"`
	CacheFlag    bool          `xml:"cacheFlag"`
	Rating       byte          `xml:"rating"`

	Values ArrayOfRecordField `xml:"values"`
}

// Combined struct for all of the different value types:
// ByteValue, ShortValue, IntValue, FloatValue, AsciiStringValue
// UnicodeStringValue, BooleanValue, DateAndTimeValue, BinaryDataValue
// Int64Value
type CommonValue struct {
	XMLName xml.Name
	Value   string `xml:"value"`
}

type StorageResponseEnvelope struct {
	XMLName       xml.Name            `xml:"s:Envelope"`
	NamespaceSoap string              `xml:"xmlns:s,attr"`
	Body          StorageResponseBody `xml:"s:Body"`
}

type StorageResponseBody struct {
	CreateRecordResponse     *CreateRecordResponse     `xml:"http://gamespy.net/sake CreateRecordResponse"`
	UpdateRecordResponse     *UpdateRecordResponse     `xml:"http://gamespy.net/sake UpdateRecordResponse"`
	GetMyRecordsResponse     *GetMyRecordsResponse     `xml:"http://gamespy.net/sake GetMyRecordsResponse"`
	SearchForRecordsResponse *SearchForRecordsResponse `xml:"http://gamespy.net/sake SearchForRecordsResponse"`
}

type CreateRecordResponse struct {
	CreateRecordResult Result
	RecordID           int32 `xml:"recordid"`
}

type UpdateRecordResponse struct {
	UpdateRecordResult Result
}

type GetMyRecordsResponse struct {
	GetMyRecordsResult Result
	Values             ArrayOfArrayOfRecordValue `xml:"values"`
}

type SearchForRecordsResponse struct {
	SearchForRecordsResult Result
	Values                 ArrayOfArrayOfRecordValue `xml:"values"`
}

type RecordField struct {
	Name  string      `xml:"name"`
	Value RecordValue `xml:"value"`
}

type RecordValue struct {
	Value CommonValue `xml:",any"`
}

type ArrayOfString struct {
	String []string `xml:"string"`
}

type ArrayOfInt struct {
	Int []int32 `xml:"int"`
}

type ArrayOfRecordValue struct {
	RecordValues []RecordValue `xml:"RecordValue"`
}

type ArrayOfArrayOfRecordValue struct {
	ArrayOfRecordValue []ArrayOfRecordValue `xml:"ArrayOfRecordValue"`
}

type ArrayOfRecordField struct {
	RecordFields []RecordField `xml:"RecordField"`
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

	storageRequestHandlers = map[string]func(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestCommon) StorageResponseBody{
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
		NamespaceSoap: SOAPEnvNamespace,
	}

	xmlName := soap.Body.Data.XMLName.Space + "/" + soap.Body.Data.XMLName.Local
	if headerAction == xmlName || headerAction == `"`+xmlName+`"` {
		logging.Info(moduleName, "SOAPAction:", aurora.Yellow(soap.Body.Data.XMLName.Local))

		handler, ok := storageRequestHandlers[xmlName]
		if !ok {
			panic("unknown SOAPAction: " + aurora.Cyan(xmlName).String())
		}

		profileId, gameInfo, result := getRequestIdentity(moduleName, soap.Body.Data)
		if result != ResultSuccess {
			logging.Error(moduleName, "Failed to get request identity:", aurora.Cyan(result))
			response.Body.setResultTag(xmlName, result)
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

	logging.Info(moduleName, "Responding with body:", aurora.Cyan(string(out)))

	payload := append([]byte(xml.Header), out...)

	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Write(payload)
}

func getRequestIdentity(moduleName string, request StorageRequestCommon) (uint32, common.GameInfo, Result) {
	gameInfo := common.GetGameInfoByID(int(request.GameID))
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

func createRecord(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestCommon) StorageResponseBody {
	if reached, err := db.IsMaxSakeRecordsReached(profileId, MaxSakeRecordsPerProfile); err != nil {
		logging.Error(moduleName, "Failed to check max sake records:", err)
		return StorageResponseBody{CreateRecordResponse: &CreateRecordResponse{
			CreateRecordResult: ResultDatabaseUnavailable,
		}}
	} else if reached {
		logging.Error(moduleName, "Profile", aurora.Cyan(profileId), "has reached the maximum number of sake records")
		return StorageResponseBody{CreateRecordResponse: &CreateRecordResponse{
			CreateRecordResult: ResultRecordLimitReached,
		}}
	}

	if request.TableID == "" {
		logging.Error(moduleName, "No table ID provided")
		return StorageResponseBody{CreateRecordResponse: &CreateRecordResponse{
			CreateRecordResult: ResultTableNotFound,
		}}
	}

	table := GetTable(gameInfo.Name, request.TableID)
	if table != nil && table.Reserved {
		// Reserved for special handler
		logging.Error(moduleName, "Attempt to create record in reserved table", aurora.Cyan(request.TableID), "in game", aurora.BrightCyan(gameInfo.Name))
		return StorageResponseBody{CreateRecordResponse: &CreateRecordResponse{
			CreateRecordResult: ResultNoPermission,
		}}
	}

	if !table.AllowsPublicCreate() {
		logging.Error(moduleName, "Attempt to create record in table that doesn't allow public create", aurora.Cyan(request.TableID), "in game", aurora.BrightCyan(gameInfo.Name))
		return StorageResponseBody{CreateRecordResponse: &CreateRecordResponse{
			CreateRecordResult: ResultNoPermission,
		}}
	}

	var record database.SakeRecord
	var result Result
	record.Fields, result = getInputFields(moduleName, request, table, true)
	if result != ResultSuccess {
		return StorageResponseBody{CreateRecordResponse: &CreateRecordResponse{
			CreateRecordResult: result,
		}}
	}

	record.GameId = gameInfo.GameID
	record.TableId = request.TableID
	record.RecordId = 0
	record.OwnerId = int32(profileId)

	// TODO: Limit number of records or fields a user can have
	recordId, err := db.InsertSakeRecord(record)
	if err != nil {
		logging.Error(moduleName, "Failed to insert sake record into the database:", err)
		return StorageResponseBody{CreateRecordResponse: &CreateRecordResponse{
			CreateRecordResult: ResultDatabaseUnavailable,
		}}
	}

	logging.Info(moduleName, "Created record in table", aurora.Cyan(record.TableId), "with ID", aurora.Cyan(recordId), "for profile", aurora.Cyan(profileId))

	return StorageResponseBody{CreateRecordResponse: &CreateRecordResponse{
		CreateRecordResult: ResultSuccess,
		RecordID:           recordId,
	}}
}

func getMyRecords(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestCommon) StorageResponseBody {
	if len(request.Fields.String) == 0 {
		// GameSpy client doesn't consider zero fields valid
		return StorageResponseBody{GetMyRecordsResponse: &GetMyRecordsResponse{
			GetMyRecordsResult: "BadNumFields",
		}}
	}
	if request.TableID == "" {
		logging.Error(moduleName, "No table ID provided")
		return StorageResponseBody{GetMyRecordsResponse: &GetMyRecordsResponse{
			GetMyRecordsResult: ResultTableNotFound,
		}}
	}

	table := GetTable(gameInfo.Name, request.TableID)
	if table != nil && table.Reserved {
		logging.Error(moduleName, "Attempt to get my records from reserved table", aurora.Cyan(request.TableID))
		return StorageResponseBody{GetMyRecordsResponse: &GetMyRecordsResponse{
			GetMyRecordsResult: ResultTableNotFound,
		}}
	}

	records, err := db.GetSakeRecords(gameInfo.GameID, []int32{int32(profileId)}, request.TableID, nil, request.Fields.String, request.Filter)
	if err != nil {
		logging.Error(moduleName, "Failed to get sake records from the database:", err)
		if err == pgx.ErrNoRows {
			return StorageResponseBody{GetMyRecordsResponse: &GetMyRecordsResponse{
				GetMyRecordsResult: ResultRecordNotFound,
			}}
		}
		return StorageResponseBody{GetMyRecordsResponse: &GetMyRecordsResponse{
			GetMyRecordsResult: ResultDatabaseUnavailable,
		}}
	}

	responseValues, result := fillResponseValues(moduleName, profileId, table, records, request)
	response := GetMyRecordsResponse{
		GetMyRecordsResult: result,
		Values:             responseValues,
	}
	logging.Info(moduleName, "Returning", aurora.Cyan(len(records)), "records from table", aurora.Cyan(request.TableID), "for profile", aurora.Cyan(profileId))

	return StorageResponseBody{GetMyRecordsResponse: &response}
}

func updateRecord(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestCommon) StorageResponseBody {
	if request.TableID == "" {
		logging.Error(moduleName, "No table ID provided")
		return StorageResponseBody{UpdateRecordResponse: &UpdateRecordResponse{
			UpdateRecordResult: ResultTableNotFound,
		}}
	}

	table := GetTable(gameInfo.Name, request.TableID)
	if table != nil && table.Reserved {
		// Reserved for special handler
		logging.Error(moduleName, "Attempt to update record in reserved table", aurora.Cyan(request.TableID), "in game", aurora.BrightCyan(gameInfo.Name))
		return StorageResponseBody{UpdateRecordResponse: &UpdateRecordResponse{
			UpdateRecordResult: ResultNoPermission,
		}}
	}

	if !table.AllowsOwnerUpdate() {
		logging.Error(moduleName, "Attempt to update record in table that doesn't allow owner update", aurora.Cyan(request.TableID), "in game", aurora.BrightCyan(gameInfo.Name))
		return StorageResponseBody{UpdateRecordResponse: &UpdateRecordResponse{
			UpdateRecordResult: ResultNoPermission,
		}}
	}

	var record database.SakeRecord
	var result Result
	record.Fields, result = getInputFields(moduleName, request, table, false)
	if result != ResultSuccess {
		return StorageResponseBody{UpdateRecordResponse: &UpdateRecordResponse{
			UpdateRecordResult: result,
		}}
	}

	record.GameId = gameInfo.GameID
	record.TableId = request.TableID
	record.RecordId = int32(request.RecordID)
	record.OwnerId = int32(profileId)

	err := db.UpdateSakeRecord(record, int32(profileId))
	if err != nil {
		logging.Error(moduleName, "Failed to update sake record in the database:", err)
		if err == database.ErrSakeNotOwned {
			return StorageResponseBody{UpdateRecordResponse: &UpdateRecordResponse{
				UpdateRecordResult: ResultNotOwned,
			}}
		}
		if err == pgx.ErrNoRows {
			return StorageResponseBody{UpdateRecordResponse: &UpdateRecordResponse{
				UpdateRecordResult: ResultRecordNotFound,
			}}
		}
		return StorageResponseBody{UpdateRecordResponse: &UpdateRecordResponse{
			UpdateRecordResult: ResultDatabaseUnavailable,
		}}
	}

	logging.Info(moduleName, "Updated record", aurora.Cyan(record.RecordId), "in table", aurora.Cyan(record.TableId), "for profile", aurora.Cyan(profileId))

	return StorageResponseBody{UpdateRecordResponse: &UpdateRecordResponse{
		UpdateRecordResult: ResultSuccess,
	}}
}

func searchForRecords(moduleName string, profileId uint32, gameInfo common.GameInfo, request StorageRequestCommon) StorageResponseBody {
	var records []database.SakeRecord

	if request.TableID == "" {
		logging.Error(moduleName, "No table ID provided")
		return StorageResponseBody{SearchForRecordsResponse: &SearchForRecordsResponse{
			SearchForRecordsResult: ResultTableNotFound,
		}}
	}

	table := GetTable(gameInfo.Name, request.TableID)

	if table != nil && table.SearchForRecordsHandler != nil {
		var ok bool
		records, ok = table.SearchForRecordsHandler(moduleName, request)
		if !ok {
			return StorageResponseBody{SearchForRecordsResponse: &SearchForRecordsResponse{
				SearchForRecordsResult: ResultUnknownError,
			}}
		}
	} else if table != nil && table.Reserved {
		logging.Error(moduleName, "Attempt to search for records in reserved table", aurora.Cyan(request.TableID))
		return StorageResponseBody{SearchForRecordsResponse: &SearchForRecordsResponse{
			SearchForRecordsResult: ResultTableNotFound,
		}}
	} else {
		ownerIds := request.OwnerIDs.Int
		if !table.AllowsPublicRead() {
			ownerIds = []int32{int32(profileId)}
		}

		var err error
		records, err = db.GetSakeRecords(gameInfo.GameID, ownerIds, request.TableID, nil, request.Fields.String, request.Filter)
		if err != nil {
			logging.Error(moduleName, "Failed to get sake records from the database:", err)
			return StorageResponseBody{SearchForRecordsResponse: &SearchForRecordsResponse{
				SearchForRecordsResult: ResultDatabaseUnavailable,
			}}
		}
	}

	// Sort the records now. TODO: This can be done more effectively in the database query.
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
	if request.Max > 0 && len(records) > int(request.Max) {
		records = records[:request.Max]
	}

	responseValues, result := fillResponseValues(moduleName, profileId, table, records, request)
	response := SearchForRecordsResponse{
		SearchForRecordsResult: result,
		Values:                 responseValues,
	}
	logging.Info(moduleName, "Found", aurora.Cyan(len(records)), "records in table", aurora.Cyan(request.TableID), "for profile", aurora.Cyan(profileId), "with filter", aurora.Cyan(request.Filter))

	return StorageResponseBody{SearchForRecordsResponse: &response}
}

func getInputFields(moduleName string, request StorageRequestCommon, table *SakeTable, useDefault bool) (map[string]database.SakeField, Result) {
	if len(request.Values.RecordFields) > MaxSakeFieldsPerRecord {
		logging.Error(moduleName, "Too many fields in record:", aurora.Cyan(len(request.Values.RecordFields)))
		return nil, ResultFieldTypeInvalid
	}
	var fields map[string]database.SakeField
	if useDefault {
		fields = table.GetDefaultFields()
	} else {
		fields = make(map[string]database.SakeField)
	}

	for _, field := range request.Values.RecordFields {
		value := field.Value.Value.Value
		fieldType, ok := tagToSakeType[field.Value.Value.XMLName.Local]
		if !ok {
			logging.Error(moduleName, "Invalid field type tag:", aurora.Cyan(field.Value.Value.XMLName.Local))
			return nil, ResultFieldTypeInvalid
		}
		if field.Name == "ownerid" || field.Name == "recordid" || field.Name == "gameid" || field.Name == "tableid" {
			logging.Error(moduleName, "Attempt to set reserved field:", aurora.Cyan(field.Name))
			return nil, ResultNoPermission
		}
		sakeField := database.SakeField{
			Type:  fieldType,
			Value: value,
		}
		if result := table.CheckValidField(field.Name, sakeField); result != ResultSuccess {
			logging.Error(moduleName, "Invalid value for field", aurora.Cyan(field.Name).String()+":", aurora.Cyan(value))
			return nil, result
		}
		var result Result
		sakeField.Value, result = table.FilterFieldFromClient(field.Name, value)
		if result != ResultSuccess {
			logging.Error(moduleName, "Failed to filter from client value for field", aurora.Cyan(field.Name), ":", aurora.Cyan(result))
			return nil, result
		}
		fields[field.Name] = sakeField
	}
	return fields, ResultSuccess
}

func fillResponseValues(moduleName string, profileId uint32, table *SakeTable, records []database.SakeRecord, request StorageRequestCommon) (ArrayOfArrayOfRecordValue, Result) {
	var response ArrayOfArrayOfRecordValue
	for _, record := range records {
		valueArray := ArrayOfRecordValue{}
		for _, field := range request.Fields.String {
			if field == "ownerid" {
				valueArray.RecordValues = append(valueArray.RecordValues, RecordValue{Value: CommonValue{
					XMLName: xml.Name{Local: "intValue"},
					Value:   strconv.FormatInt(int64(int32(record.OwnerId)), 10),
				}})
				continue
			}
			if field == "recordid" {
				valueArray.RecordValues = append(valueArray.RecordValues, RecordValue{Value: CommonValue{
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
				logging.Warn(moduleName, "Field", aurora.Cyan(field), "not found in record", aurora.Cyan(record.RecordId))
				valueArray = ArrayOfRecordValue{}
				break
			}

			var result Result
			fieldValue.Value, result = table.FilterFieldFromDatabase(field, fieldValue.Value, record.OwnerId == int32(profileId))
			if result != ResultSuccess {
				logging.Error(moduleName, "Failed to filter to client value for field", aurora.Cyan(field), ":", aurora.Cyan(result))
				return ArrayOfArrayOfRecordValue{}, result
			}
			value := fillValue(fieldValue.Type, fieldValue.Value)
			valueArray.RecordValues = append(valueArray.RecordValues, RecordValue{Value: value})
		}
		if len(valueArray.RecordValues) == 0 {
			continue
		}
		response.ArrayOfRecordValue = append(response.ArrayOfRecordValue, valueArray)
	}
	return response, ResultSuccess
}

func fillValue(valueType database.SakeFieldType, value string) CommonValue {
	return CommonValue{
		XMLName: xml.Name{Local: sakeTypeToTag[valueType]},
		Value:   value,
	}
}

func (body *StorageResponseBody) setResultTag(xmlName string, result Result) {
	switch xmlName {
	case SakeNamespace + "/CreateRecord":
		body.CreateRecordResponse = &CreateRecordResponse{
			CreateRecordResult: result,
		}
	case SakeNamespace + "/UpdateRecord":
		body.UpdateRecordResponse = &UpdateRecordResponse{
			UpdateRecordResult: result,
		}
	case SakeNamespace + "/GetMyRecords":
		body.GetMyRecordsResponse = &GetMyRecordsResponse{
			GetMyRecordsResult: result,
		}
	case SakeNamespace + "/SearchForRecords":
		body.SearchForRecordsResponse = &SearchForRecordsResponse{}
	}
}
