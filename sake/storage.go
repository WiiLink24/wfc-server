package sake

import (
	"encoding/base64"
	"encoding/xml"
	"github.com/logrusorgru/aurora/v3"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"wwfc/database"
	"wwfc/logging"
)

type StorageRequestEnvelope struct {
	XMLName xml.Name
	Body    StorageRequestBody
}

type StorageRequestBody struct {
	XMLName          xml.Name
	GetMyRecords     StorageGetMyRecords
	UpdateRecord     StorageUpdateRecord
	SearchForRecords StorageSearchForRecords
}

type StorageGetMyRecords struct {
	XMLName     xml.Name
	GameID      int           `xml:"gameid"`
	SecretKey   string        `xml:"secretKey"`
	LoginTicket string        `xml:"loginTicket"`
	TableID     string        `xml:"tableid"`
	Fields      StorageFields `xml:"fields"`
}

type StorageFields struct {
	XMLName xml.Name
	Fields  []string `xml:"string"`
}

type StorageUpdateRecord struct {
	XMLName     xml.Name
	GameID      int                       `xml:"gameid"`
	SecretKey   string                    `xml:"secretKey"`
	LoginTicket string                    `xml:"loginTicket"`
	TableID     string                    `xml:"tableid"`
	RecordID    string                    `xml:"recordid"`
	Values      StorageUpdateRecordValues `xml:"values"`
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

type StorageSearchForRecords struct {
	XMLName     xml.Name
	GameID      int           `xml:"gameid"`
	SecretKey   string        `xml:"secretKey"`
	LoginTicket string        `xml:"loginTicket"`
	TableID     string        `xml:"tableid"`
	Filter      string        `xml:"filter"`
	Sort        string        `xml:"sort"`
	Offset      int           `xml:"offset"`
	Max         int           `xml:"max"`
	Surrounding int           `xml:"surrounding"`
	OwnerIDs    string        `xml:"ownerids"`
	CacheFlag   int           `xml:"cacheFlag"`
	Fields      StorageFields `xml:"fields"`
}

type StorageResponseEnvelope struct {
	XMLName xml.Name
	Body    StorageResponseBody
}

type StorageResponseBody struct {
	XMLName                  xml.Name
	GetMyRecordsResponse     *StorageGetMyRecordsResponse
	UpdateRecordResponse     *StorageUpdateRecordResponse
	SearchForRecordsResponse *StorageSearchForRecordsResponse
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

func handleStorageRequest(moduleName string, w http.ResponseWriter, r *http.Request) {
	action := r.Header.Get("SOAPAction")
	if action == "" {
		logging.Error(moduleName, "No SOAPAction in header")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	// logging.Notice(moduleName, string(body))

	// Parse the SOAP request XML
	soap := StorageRequestEnvelope{}
	err = xml.Unmarshal(body, &soap)
	if err != nil {
		panic(err)
	}

	response := StorageResponseEnvelope{
		XMLName: xml.Name{"http://schemas.xmlsoap.org/soap/envelope/", "Envelope"},
		Body: StorageResponseBody{
			XMLName: xml.Name{"http://schemas.xmlsoap.org/soap/envelope/", "Body"},
		},
	}

	switch action {
	case `"http://gamespy.net/sake/GetMyRecords"`:
		response.Body.GetMyRecordsResponse = getMyRecords(moduleName, soap.Body.GetMyRecords)
		break

	case `"http://gamespy.net/sake/UpdateRecord"`:
		response.Body.UpdateRecordResponse = updateRecord(moduleName, soap.Body.UpdateRecord)
		break

	case `"http://gamespy.net/sake/SearchForRecords"`:
		response.Body.SearchForRecordsResponse = searchForRecords(moduleName, soap.Body.SearchForRecords)
		break

	default:
		logging.Error(moduleName, "Unknown SOAPAction:", aurora.Cyan(action))
		return
	}

	out, err := xml.Marshal(response)
	if err != nil {
		panic(err)
	}

	payload := append([]byte(`<?xml version="1.0" encoding="utf-8"?>`), out...)
	// logging.Notice(moduleName, string(payload))

	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Write(payload)
}

func binaryDataValue(value []byte) StorageValue {
	return StorageValue{
		XMLName: xml.Name{"", "binaryDataValue"},
		Value:   base64.StdEncoding.EncodeToString([]byte(value)),
	}
}

func binaryDataValueBase64(value string) StorageValue {
	return StorageValue{
		XMLName: xml.Name{"", "binaryDataValue"},
		Value:   value,
	}
}

func intValue(value int32) StorageValue {
	return StorageValue{
		XMLName: xml.Name{"", "intValue"},
		Value:   strconv.FormatInt(int64(value), 10),
	}
}

// I don't even know if this is a thing
func uintValue(value uint32) StorageValue {
	return StorageValue{
		XMLName: xml.Name{"", "uintValue"},
		Value:   strconv.FormatUint(uint64(value), 10),
	}
}

func getMyRecords(moduleName string, request StorageGetMyRecords) *StorageGetMyRecordsResponse {
	if request.XMLName.Space+"/"+request.XMLName.Local != "http://gamespy.net/sake/GetMyRecords" {
		logging.Error(moduleName, "Missing GetMyRecords in request")
		return nil
	}

	logging.Notice(moduleName, "SOAPAction:", aurora.Yellow("GetMyRecords"))

	_, profileId := database.GetSession(pool, ctx, request.LoginTicket)
	logging.Info(moduleName, "Profile ID:", aurora.BrightCyan(profileId))
	logging.Info(moduleName, "Game ID:", aurora.Cyan(request.GameID))
	logging.Info(moduleName, "Table ID:", aurora.Cyan(request.TableID))

	errorResponse := StorageGetMyRecordsResponse{
		XMLName:            xml.Name{"http://gamespy.net/sake", "GetMyRecordsResponse"},
		GetMyRecordsResult: "Error",
	}

	values := map[string]StorageValue{}

	switch strconv.Itoa(request.GameID) + "/" + request.TableID {
	default:
		logging.Error(moduleName, "Unknown table")
		return &errorResponse

	case "1687/FriendInfo":
		// Mario Kart Wii friend info
		values = map[string]StorageValue{
			"ownerid":  uintValue(uint32(profileId)),
			"recordid": intValue(int32(profileId)),
			"info":     binaryDataValueBase64(database.GetMKWFriendInfo(pool, ctx, profileId)),
		}
		break
	}

	response := StorageGetMyRecordsResponse{
		XMLName:            xml.Name{"http://gamespy.net/sake", "GetMyRecordsResponse"},
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

	logging.Notice(moduleName, "Wrote", aurora.Cyan(fieldCount), "field(s)")
	return &response
}

func updateRecord(moduleName string, request StorageUpdateRecord) *StorageUpdateRecordResponse {
	if request.XMLName.Space+"/"+request.XMLName.Local != "http://gamespy.net/sake/UpdateRecord" {
		logging.Error(moduleName, "Missing UpdateRecord in request")
		return nil
	}

	logging.Notice(moduleName, "SOAPAction:", aurora.Yellow("UpdateRecord"))

	_, profileId := database.GetSession(pool, ctx, request.LoginTicket)
	logging.Info(moduleName, "Profile ID:", aurora.BrightCyan(profileId))
	logging.Info(moduleName, "Game ID:", aurora.Cyan(request.GameID))
	logging.Info(moduleName, "Table ID:", aurora.Cyan(request.TableID))

	errorResponse := StorageUpdateRecordResponse{
		XMLName:            xml.Name{"http://gamespy.net/sake", "UpdateRecordResponse"},
		UpdateRecordResult: "Error",
	}

	switch strconv.Itoa(request.GameID) + "/" + request.TableID {
	default:
		logging.Error(moduleName, "Unknown table")
		return &errorResponse

	case "1687/FriendInfo":
		// Mario Kart Wii friend info
		if len(request.Values.RecordFields) != 1 || request.Values.RecordFields[0].Name != "info" || request.Values.RecordFields[0].Value.Value.XMLName.Local != "binaryDataValue" {
			logging.Error(moduleName, "Invalid record fields")
			return &errorResponse
		}

		// TODO: Validate record data
		database.UpdateMKWFriendInfo(pool, ctx, profileId, request.Values.RecordFields[0].Value.Value.Value)
		logging.Notice(moduleName, "Updated Mario Kart Wii friend info")
		break
	}

	return &StorageUpdateRecordResponse{
		XMLName:            xml.Name{"http://gamespy.net/sake", "UpdateRecordResponse"},
		UpdateRecordResult: "Success",
	}
}

func searchForRecords(moduleName string, request StorageSearchForRecords) *StorageSearchForRecordsResponse {
	if request.XMLName.Space+"/"+request.XMLName.Local != "http://gamespy.net/sake/SearchForRecords" {
		logging.Error(moduleName, "Missing SearchForRecords in request")
		return nil
	}

	logging.Notice(moduleName, "SOAPAction:", aurora.Yellow("SearchForRecords"))

	_, profileId := database.GetSession(pool, ctx, request.LoginTicket)
	logging.Info(moduleName, "Profile ID:", aurora.BrightCyan(profileId))
	logging.Info(moduleName, "Game ID:", aurora.Cyan(request.GameID))
	logging.Info(moduleName, "Table ID:", aurora.Cyan(request.TableID))
	logging.Info(moduleName, "Filter:", aurora.Cyan(request.Filter))

	errorResponse := StorageSearchForRecordsResponse{
		XMLName:                xml.Name{"http://gamespy.net/sake", "SearchForRecordsResponse"},
		SearchForRecordsResult: "Error",
	}

	values := []map[string]StorageValue{}

	switch strconv.Itoa(request.GameID) + "/" + request.TableID {
	default:
		logging.Error(moduleName, "Unknown table")
		return &errorResponse

	case "1687/FriendInfo":
		// Mario Kart Wii friend info
		match := regexp.MustCompile(`^ownerid = (\d{1,10})$`).FindStringSubmatch(request.Filter)
		if len(match) != 2 {
			logging.Error(moduleName, "Invalid filter")
			return &errorResponse
		}

		ownerId, err := strconv.ParseUint(match[1], 10, 32)
		if err != nil {
			logging.Error(moduleName, "Invalid owner ID")
			return &errorResponse
		}

		// TODO: Check if the two are friends maybe

		values = []map[string]StorageValue{
			map[string]StorageValue{
				"ownerid":  uintValue(uint32(ownerId)),
				"recordid": intValue(int32(ownerId)),
				"info":     binaryDataValueBase64(database.GetMKWFriendInfo(pool, ctx, uint32(ownerId))),
			},
		}
		break
	}

	// Sort the values now
	sort.Slice(values, func(l, r int) bool {
		lVal, lExists := values[l][request.Sort]
		rVal, rExists := values[r][request.Sort]
		if lExists == false || rExists == false {
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
		XMLName:                xml.Name{"http://gamespy.net/sake", "SearchForRecordsResponse"},
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

	logging.Notice(moduleName, "Wrote", aurora.BrightCyan(fieldCount), "field(s) across", aurora.BrightCyan(i), "record(s)")
	return &response
}
