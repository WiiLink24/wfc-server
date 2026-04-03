package sake

const GameSpyMultipartBoundary = "Qr4G823s23d---<<><><<<>--7d118e0536"

type SakeFileResult int

// https://documentation.help/GameSpy-SDK/SAKEFileResult.html
const (
	SakeFileResultHeader = "Sake-File-Result"

	SakeFileResultSuccess          = 0
	SakeFileResultMissingParameter = 3
	SakeFileResultFileNotFound     = 4
	SakeFileResultFileTooLarge     = 5
	SakeFileResultServerError      = 6
)

const (
	ResultSuccess             = "Success"             // 4xx51
	ResultSecretKeyInvalid    = "SecretKeyInvalid"    // 4xx52
	ResultServiceDisabled     = "ServiceDisabled"     // 4xx53
	ResultDatabaseUnavailable = "DatabaseUnavailable" // 4xx58
	ResultLoginTicketInvalid  = "LoginTicketInvalid"  // 4xx59
	ResultLoginTicketExpired  = "LoginTicketExpired"  // 4xx60
	ResultTableNotFound       = "TableNotFound"       // 4xx61
	ResultRecordNotFound      = "RecordNotFound"      // 4xx62
	ResultFieldNotFound       = "FieldNotFound"       // 4xx63
	ResultFieldTypeInvalid    = "FieldTypeInvalid"    // 4xx64
	ResultNoPermission        = "NoPermission"        // 4xx65
	ResultRecordLimitReached  = "RecordLimitReached"  // 4xx66
	ResultAlreadyRated        = "AlreadyRated"        // 4xx67
	ResultNotRateable         = "NotRateable"         // 4xx68
	ResultNotOwned            = "NotOwned"            // 4xx69
	ResultFilterInvalid       = "FilterInvalid"       // 4xx70
	ResultSortInvalid         = "SortInvalid"         // 4xx71
	ResultTargetFilterInvalid = "TargetFilterInvalid" // 4xx80
	ResultUnknownError        = "UnknownError"        // 4xx72
)
