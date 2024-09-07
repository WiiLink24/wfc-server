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
