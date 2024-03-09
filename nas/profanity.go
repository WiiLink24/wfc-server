package nas

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

var profanityFilePath = "./profanity.txt"
var profanityFileLines []string = nil

func CacheProfanityFile() error {
	file, err := os.Open(profanityFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		profanityFileLines = append(profanityFileLines, line)
	}

	if profanityFileLines == nil {
		return errors.New("the file '" + profanityFilePath + "' is empty")
	}

	return nil
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
