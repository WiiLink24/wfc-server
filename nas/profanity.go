package nas

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"time"
)

var profanityFilePath = "./profanity.txt"
var profanityFileLines []string = nil
var lastModTime time.Time

var symbolEquivalences = map[rune]rune{
    '1': 'i',
    '0': 'o',
    '5': 's',
    '4': 'a',
    '3': 'e',
    '7': 't',
    '9': 'g',
    '2': 'z',
    '(': 'c',
}

func CacheProfanityFile() error {
	fileInfo, err := os.Stat(profanityFilePath)
	if err != nil {
		return err
	}

	if !fileInfo.ModTime().After(lastModTime) && profanityFileLines != nil {
		return nil
	}

	file, err := os.Open(profanityFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	profanityFileLines = nil
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

	lastModTime = fileInfo.ModTime()
	return nil
}

func normalizeWord(word string) string {
	var normalized strings.Builder
	for _, char := range word {
		if equivalent, exists := symbolEquivalences[char]; exists {
			normalized.WriteRune(equivalent)
		} else {
			normalized.WriteRune(char)
		}
	}
	return normalized.String()
}

func IsBadWord(word string) (bool, error) {
	if !isProfanityFileCached() {
		err := CacheProfanityFile()
		if err != nil {
			return false, errors.New("the file '" + profanityFilePath + "' has not been cached")
		}

	}

	normalizedWord := normalizeWord(word)
	for _, line := range profanityFileLines {
		if strings.EqualFold(line, normalizedWord) {
			return true, nil
		}
	}

	return false, nil
}

func isProfanityFileCached() bool {
	fileInfo, err := os.Stat(profanityFilePath)
	if err != nil {
		return false
	}
	return profanityFileLines != nil && !fileInfo.ModTime().After(lastModTime)
}
