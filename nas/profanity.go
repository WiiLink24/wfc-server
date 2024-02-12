package nas

import (
	"errors"
	"os"
	"strings"
)

var profanityFilePath = "./profanity.txt"
var profanityFileLines []string = nil

func CacheProfanityFile() bool {
	contents, err := os.ReadFile(profanityFilePath)
	if err != nil {
		return false
	}

	lines := strings.Split(string(contents), "\n")
	profanityFileLines = lines
	return true
}

func IsBadWord(word string) (bool, error) {
	if !isProfanityFileCached() {
		return false, errors.New("the file '" + profanityFilePath + "' has not been cached")
	}

	for _, line := range profanityFileLines {
		if strings.EqualFold(line, word) {
			return true, nil
		}
	}

	return false, nil
}

func isProfanityFileCached() bool {
	return profanityFileLines != nil
}
