package sake

import (
	"encoding/base64"
	"strconv"
	"time"
	"wwfc/database"
)

const (
	MaxSakeRecordsPerProfile = 96
	MaxSakeFieldsPerRecord   = 64
	MaxSakeFieldValueLength  = 4096
)

const DateAndTimeFormat = "2006-01-02T15:04:05.000"

const (
	RateableUnknown = iota
	RateableYes
	RateableNo
)

type Rateable byte

const (
	OwnerTypeProfile = iota
	OwnerTypeBackend
)

type OwnerType byte

const (
	PermissionDefault = iota
	PermissionAllowed
	PermissionDenied
)

type Permission byte

type SakeFieldDefinition struct {
	// Type of the field.
	Type database.SakeFieldType
	// If empty, no default value will be set.
	// "{EMPTY}" can be used to set an empty string default value.
	// "{CURRENT_TIMESTAMP}" can be used to set the current timestamp as the default value (only for DateAndTime type).
	Default string
	// If zero, the default field length limit will be used.
	LengthLimit int

	// Optional function for custom validation.
	IsValidFunc func(value string) bool
	// Optional function for custom filtering. This function receives the value from the client AFTER validation, before inserting into the database.
	FilterFromClientFunc func(value string, isOwner bool) (string, string)
	// Optional function for custom filtering. This function receives the value from the database before sending to the client.
	FilterFromDatabaseFunc func(value string, isOwner bool) (string, string)
}

type SakeTable struct {
	// Determines whether the 'average_rating', 'my_rating', 'num_ratings', 'sum_ratings' fields are automatically added.
	Rateable Rateable
	// Defaults to profile-owned records if not specified.
	OwnerType OwnerType
	// Defaults to allowed if OwnerType is OwnerTypeProfile, denied otherwise.
	PublicPermCreate Permission
	// Defaults to allowed.
	PublicPermRead Permission
	// Defaults to allowed.
	OwnerPermUpdate Permission
	// Defaults to allowed.
	OwnerPermDelete Permission
	// Override the default maximum number of records per owner.
	LimitPerOwner int
	// If true, fields not specified in this table definition will be rejected.
	Hardened bool
	// If true, Sake will return a NoPermission error for requests that don't have a custom handler
	Reserved bool
	// Custom handler for SearchForRecords. Returns an array of response Sake records.
	SearchForRecordsHandler func(string, StorageRequestData) ([]database.SakeRecord, bool)
	// Field definitions for this table. The key is the field name.
	Fields map[string]SakeFieldDefinition
}

var TableDefinitions = map[string]SakeTable{
	"micchannelwii/userinfo": {
		Rateable:         RateableYes,
		OwnerType:        OwnerTypeProfile,
		PublicPermCreate: PermissionAllowed,
		PublicPermRead:   PermissionAllowed,
		OwnerPermUpdate:  PermissionAllowed,
		OwnerPermDelete:  PermissionAllowed,
		Fields: map[string]SakeFieldDefinition{
			"wiiid": {
				Type: database.SakeFieldTypeInt64,
			},
			"username": {
				Type: database.SakeFieldTypeBinaryData,
			},
			"friendkey": {
				Type: database.SakeFieldTypeInt64,
			},
		},
	},

	"mariokartwii/FriendInfo": {
		Rateable:         RateableNo,
		OwnerType:        OwnerTypeProfile,
		PublicPermCreate: PermissionAllowed,
		PublicPermRead:   PermissionAllowed,
		OwnerPermUpdate:  PermissionAllowed,
		OwnerPermDelete:  PermissionAllowed,
		LimitPerOwner:    1,
		Hardened:         false, // To allow modding extra fields
		Fields: map[string]SakeFieldDefinition{
			"info": {
				Type:                   database.SakeFieldTypeBinaryData,
				FilterFromDatabaseFunc: filterMarioKartWiiFriendInfo,
			},
		},
	},
	"mariokartwii/GhostData": {
		Rateable:                RateableNo,
		OwnerType:               OwnerTypeProfile,
		PublicPermCreate:        PermissionAllowed,
		PublicPermRead:          PermissionAllowed,
		OwnerPermUpdate:         PermissionAllowed,
		OwnerPermDelete:         PermissionAllowed,
		Hardened:                true,
		Reserved:                true,
		SearchForRecordsHandler: getMarioKartWiiStoredGhostDataRecord,
		Fields: map[string]SakeFieldDefinition{
			"fileid": {
				Type: database.SakeFieldTypeInt,
			},
			"profile": {
				Type: database.SakeFieldTypeInt,
			},
			"course": {
				Type: database.SakeFieldTypeInt,
			},
			"region": {
				Type: database.SakeFieldTypeInt,
			},
			"time": {
				Type: database.SakeFieldTypeInt,
			},
		},
	},
	"mariokartwii/StoredGhostData": {
		Rateable:                RateableNo,
		OwnerType:               OwnerTypeProfile,
		PublicPermCreate:        PermissionAllowed,
		PublicPermRead:          PermissionAllowed,
		OwnerPermUpdate:         PermissionAllowed,
		OwnerPermDelete:         PermissionAllowed,
		Hardened:                true,
		Reserved:                true,
		SearchForRecordsHandler: getMarioKartWiiStoredGhostDataRecord,
		Fields: map[string]SakeFieldDefinition{
			"fileid": {
				Type: database.SakeFieldTypeInt,
			},
			"profile": {
				Type: database.SakeFieldTypeInt,
			},
			"course": {
				Type: database.SakeFieldTypeInt,
			},
			"region": {
				Type: database.SakeFieldTypeInt,
			},
			"time": {
				Type: database.SakeFieldTypeInt,
			},
		},
	},

	"guinnesswrds/RecordTable": {
		Rateable:         RateableUnknown,
		OwnerType:        OwnerTypeProfile,
		PublicPermCreate: PermissionAllowed,
		PublicPermRead:   PermissionAllowed,
		OwnerPermUpdate:  PermissionAllowed,
		OwnerPermDelete:  PermissionAllowed,
		Fields: map[string]SakeFieldDefinition{
			"Score": {
				Type: database.SakeFieldTypeInt,
			},
			"GameID": {
				Type: database.SakeFieldTypeByte,
			},
			"Region": {
				Type: database.SakeFieldTypeByte,
			},
			"Country": {
				Type: database.SakeFieldTypeByte,
			},
			"OwnerName": {
				Type: database.SakeFieldTypeUnicodeString,
			},
			"AvatarName": {
				Type: database.SakeFieldTypeUnicodeString,
			},
			"AvatarModel": {
				Type: database.SakeFieldTypeByte,
			},
			"AvatarParts": {
				Type: database.SakeFieldTypeBinaryData,
			},
			"DateTimeSet": {
				Type:    database.SakeFieldTypeDateAndTime,
				Default: "{CURRENT_TIMESTAMP}",
			},
		},
	},
}

func GetTable(gameName string, tableId string) *SakeTable {
	tableDef, exists := TableDefinitions[gameName+"/"+tableId]
	if !exists {
		return nil
	}
	return &tableDef
}

func (t *SakeTable) AllowsPublicCreate() bool {
	if t == nil {
		return true
	}
	if t.PublicPermCreate == PermissionDefault {
		return t.OwnerType == OwnerTypeProfile
	}
	return t.PublicPermCreate == PermissionAllowed
}

func (t *SakeTable) AllowsPublicRead() bool {
	if t == nil {
		return true
	}
	return t.PublicPermRead == PermissionAllowed || t.PublicPermRead == PermissionDefault
}

func (t *SakeTable) AllowsOwnerUpdate() bool {
	if t == nil {
		return true
	}
	return t.OwnerPermUpdate == PermissionAllowed || t.OwnerPermUpdate == PermissionDefault
}

func (t *SakeTable) AllowsOwnerDelete() bool {
	if t == nil {
		return true
	}
	return t.OwnerPermDelete == PermissionAllowed || t.OwnerPermDelete == PermissionDefault
}

func (t *SakeTable) GetDefaultFields() map[string]database.SakeField {
	if t == nil {
		return nil
	}
	defaultFields := make(map[string]database.SakeField)
	for fieldName, fieldDef := range t.Fields {
		if fieldDef.Default != "" {
			value := fieldDef.Default
			if value == "{CURRENT_TIMESTAMP}" && fieldDef.Type == database.SakeFieldTypeDateAndTime {
				value = time.Now().UTC().Format(DateAndTimeFormat)
			} else if value == "{EMPTY}" {
				value = ""
			}

			defaultFields[fieldName] = database.SakeField{
				Type:  fieldDef.Type,
				Value: value,
			}
		}
	}
	if t.Rateable == RateableYes {
		defaultFields["average_rating"] = database.SakeField{
			Type:  database.SakeFieldTypeFloat,
			Value: "0",
		}
		defaultFields["num_ratings"] = database.SakeField{
			Type:  database.SakeFieldTypeInt,
			Value: "0",
		}
	}

	return defaultFields
}

func (t *SakeTable) CheckValidField(fieldName string, field database.SakeField) string {
	lengthLimit := MaxSakeFieldValueLength
	var verifyFunc func(value string) bool
	if t != nil && len(t.Fields) != 0 {
		fieldDef, exists := t.Fields[fieldName]
		if !exists {
			if t.Hardened {
				return ResultFieldNotFound
			}
		} else if fieldDef.Type != field.Type {
			return ResultFieldTypeInvalid
		}
		if fieldDef.LengthLimit > 0 {
			lengthLimit = fieldDef.LengthLimit
		}
		verifyFunc = fieldDef.IsValidFunc
	}
	if len(field.Value) > lengthLimit {
		return ResultFieldTypeInvalid
	}

	// These values may not be written to
	if fieldName == "average_rating" || fieldName == "my_rating" || fieldName == "num_ratings" || fieldName == "sum_ratings" {
		return ResultFieldNotFound
	}

	switch field.Type {
	case database.SakeFieldTypeByte:
		if len(field.Value) == 0 || len(field.Value) > 3 {
			return ResultFieldTypeInvalid
		}
		if parsed, err := strconv.ParseUint(field.Value, 10, 8); err != nil || strconv.FormatUint(parsed, 10) != field.Value {
			return ResultFieldTypeInvalid
		}

	case database.SakeFieldTypeShort:
		if len(field.Value) == 0 || len(field.Value) > 6 {
			return ResultFieldTypeInvalid
		}
		if parsed, err := strconv.ParseInt(field.Value, 10, 16); err != nil || strconv.FormatInt(parsed, 10) != field.Value {
			return ResultFieldTypeInvalid
		}

	case database.SakeFieldTypeInt:
		if len(field.Value) == 0 || len(field.Value) > 11 {
			return ResultFieldTypeInvalid
		}
		if parsed, err := strconv.ParseInt(field.Value, 10, 32); err != nil || strconv.FormatInt(parsed, 10) != field.Value {
			return ResultFieldTypeInvalid
		}

	case database.SakeFieldTypeInt64:
		if len(field.Value) == 0 || len(field.Value) > 20 {
			return ResultFieldTypeInvalid
		}
		if parsed, err := strconv.ParseInt(field.Value, 10, 64); err != nil || strconv.FormatInt(parsed, 10) != field.Value {
			return ResultFieldTypeInvalid
		}

	case database.SakeFieldTypeFloat:
		if len(field.Value) == 0 || len(field.Value) > 24 {
			return ResultFieldTypeInvalid
		}
		_, err := strconv.ParseFloat(field.Value, 32)
		if err != nil {
			return ResultFieldTypeInvalid
		}

	case database.SakeFieldTypeBoolean:
		if field.Value != "true" && field.Value != "false" && field.Value != "1" && field.Value != "0" {
			return ResultFieldTypeInvalid
		}

	case database.SakeFieldTypeDateAndTime:
		if len(field.Value) == 0 || len(field.Value) > 24 {
			return ResultFieldTypeInvalid
		}
		_, err := time.Parse(DateAndTimeFormat, field.Value)
		if err != nil {
			return ResultFieldTypeInvalid
		}

	case database.SakeFieldTypeBinaryData:
		if len(field.Value) == 0 {
			return ResultSuccess
		}
		binaryData, err := base64.StdEncoding.Strict().DecodeString(field.Value)
		if err != nil {
			return ResultFieldTypeInvalid
		}
		if len(binaryData) > lengthLimit {
			return ResultFieldTypeInvalid
		}
	}

	if verifyFunc != nil && !verifyFunc(field.Value) {
		return ResultFieldTypeInvalid
	}
	return ResultSuccess
}

func (t *SakeTable) FilterFieldFromClient(fieldName string, value string) (string, string) {
	if t == nil || t.Fields == nil {
		return value, ResultSuccess
	}
	fieldDef, exists := t.Fields[fieldName]
	if !exists {
		return value, ResultSuccess
	}
	if fieldDef.FilterFromClientFunc == nil {
		return value, ResultSuccess
	}
	return fieldDef.FilterFromClientFunc(value, true)
}

func (t *SakeTable) FilterFieldFromDatabase(fieldName string, value string, isOwner bool) (string, string) {
	if t == nil || t.Fields == nil {
		return value, ResultSuccess
	}
	fieldDef, exists := t.Fields[fieldName]
	if !exists {
		return value, ResultSuccess
	}
	if fieldDef.FilterFromDatabaseFunc == nil {
		return value, ResultSuccess
	}
	if t.OwnerType != OwnerTypeProfile {
		isOwner = false
	}
	return fieldDef.FilterFromDatabaseFunc(value, isOwner)
}
