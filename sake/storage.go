package sake

import (
	"encoding/base64"
	"encoding/xml"
	"github.com/logrusorgru/aurora/v3"
	"io"
	"net/http"
	"strconv"
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
	Value   StorageValue `xml:",any"`
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

func intValue(value int32) StorageValue {
	return StorageValue{
		XMLName: xml.Name{"", "intValue"},
		Value:   strconv.FormatInt(int64(value), 10),
	}
}

func getRecordValue(moduleName string, tableId string, field string) (StorageRecordValue, bool) {
	// TODO: Actually implement this instead of using hardcoded values
	value := StorageValue{}

	// Temporary fixed values
	switch field {
	case "info":
		value = binaryDataValue([]byte{0xDC, 0xE6, 0x00, 0x50, 0x00, 0x61, 0x00, 0x6C, 0x00, 0x61, 0x00, 0x70, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x69, 0x00, 0x00, 0x00, 0x00, 0x52, 0x25, 0x88, 0x1B, 0xBA, 0xE4, 0x04, 0x6C, 0x0E, 0x69, 0x00, 0x04, 0x8E, 0xA0, 0x09, 0x3D, 0x26, 0x92, 0x6C, 0x8C, 0xA8, 0x40, 0x14, 0x49, 0x90, 0x4D, 0x00, 0x8A, 0x00, 0x8A, 0x25, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1A, 0x9B, 0x00, 0x08, 0x96, 0x2B, 0xFB, 0x47, 0x0E, 0x82, 0x52, 0x4D, 0x43, 0x50, 0x00, 0x00, 0x00, 0x00, 0x25, 0x59, 0x09, 0x88})
		break

	case "recordid":
		// Profile ID
		value = intValue(43)
		break

	default:
		logging.Error(moduleName, "Missing field:", aurora.Cyan(field))
		return StorageRecordValue{}, false
	}

	logging.Info(moduleName, "Field:", aurora.Cyan(field))
	return StorageRecordValue{Value: value}, true
}

func getMyRecords(moduleName string, request StorageGetMyRecords) *StorageGetMyRecordsResponse {
	if request.XMLName.Space+"/"+request.XMLName.Local != "http://gamespy.net/sake/GetMyRecords" {
		logging.Error(moduleName, "Missing GetMyRecords in request")
		return nil
	}

	logging.Notice(moduleName, "SOAPAction:", aurora.Yellow("GetMyRecords"))

	response := StorageGetMyRecordsResponse{
		XMLName:            xml.Name{"http://gamespy.net/sake", "GetMyRecordsResponse"},
		GetMyRecordsResult: "Success",
	}

	valueArray := &response.Values.ArrayOfRecordValue

	for _, field := range request.Fields.Fields {
		recordValue, ok := getRecordValue(moduleName, request.TableID, field)
		if !ok {
			// TODO: Is this actually how you return an error
			return &StorageGetMyRecordsResponse{
				XMLName:            xml.Name{"http://gamespy.net/sake", "GetMyRecordsResponse"},
				GetMyRecordsResult: "Error",
			}
		}

		valueArray.RecordValues = append(valueArray.RecordValues, recordValue)
	}

	return &response
}

func updateRecord(moduleName string, request StorageUpdateRecord) *StorageUpdateRecordResponse {
	if request.XMLName.Space+"/"+request.XMLName.Local != "http://gamespy.net/sake/UpdateRecord" {
		logging.Error(moduleName, "Missing UpdateRecord in request")
		return nil
	}

	logging.Notice(moduleName, "SOAPAction:", aurora.Yellow("UpdateRecord"))

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

	response := StorageSearchForRecordsResponse{
		XMLName:                xml.Name{"http://gamespy.net/sake", "SearchForRecordsResponse"},
		SearchForRecordsResult: "Success",
	}

	valueArray := &response.Values.ArrayOfRecordValue

	for i := 0; i < request.Max; i++ {
		for _, field := range request.Fields.Fields {
			recordValue, ok := getRecordValue(moduleName, request.TableID, field)
			if !ok {
				// TODO: Is this actually how you return an error
				return &StorageSearchForRecordsResponse{
					XMLName:                xml.Name{"http://gamespy.net/sake", "SearchForRecordsResponse"},
					SearchForRecordsResult: "Error",
				}
			}

			valueArray.RecordValues = append(valueArray.RecordValues, recordValue)
		}
	}

	return &response
}
