package nas

import (
	"bufio"
	"encoding/binary"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
	"wwfc/common"
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
	defer func() {
		common.ShouldNotError(file.Close())
	}()

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

func handleAuthProfanityEndpoint(w http.ResponseWriter, r *http.Request) {
	form, err := parseAuthRequest(r)
	if err != nil {
		replyHTTPError(w, 400, "400 Bad Request")
		return
	}

	unitcd := form["unitcd"]
	var wordsEncoding string
	var wordsDefaultEncoding string
	if len(unitcd) != 1 || unitcd[0] != '0' {
		wordsEncoding = "UTF-16BE"
		wordsDefaultEncoding = "UTF-16BE"
	} else {
		wordsEncoding = "UTF-16LE"
		wordsDefaultEncoding = "UTF-16LE"
	}

	if wencValues, ok := form["wenc"]; ok {
		// It's okay for this to error, the real server
		// just falls back to the default encoding in
		// this case even if it cant properly handle it
		wencDecoded, err := common.Base64DwcEncoding.DecodeString(string(wencValues[0]))
		if err == nil {
			wordsEncoding = string(wencDecoded)
		}
	}

	if wordsEncoding != "UTF-8" && wordsEncoding != "UTF-16LE" && wordsEncoding != "UTF-16BE" {
		wordsEncoding = wordsDefaultEncoding
	}

	// It's okay for this to not exist/be valid, the real
	// server will just treat the missing input as a single
	// non-profane word
	wordsBytes := []byte{}
	if wordsValues, ok := form["words"]; ok {
		wordsDecoded, err := common.Base64DwcEncoding.DecodeString(string(wordsValues[0]))
		if err == nil {
			wordsBytes = wordsDecoded
		}
	}

	// This field is entirely optional, unsure what
	// specifically it does. Adds extra data to the
	// reply, probably used for handling the word
	// list differently for different regions?
	var wordsRegion string
	if wordsRegionValues, ok := form["wregion"]; ok {
		wordsRegionDecoded, err := common.Base64DwcEncoding.DecodeString(string(wordsRegionValues[0]))
		if err == nil {
			wordsRegion = string(wordsRegionDecoded)
		}
	}

	var words string
	switch wordsEncoding {
	case "UTF-8":
		words = string(wordsBytes)
	case "UTF-16LE":
		words = common.UTF16Decode(wordsBytes, binary.LittleEndian)
	case "UTF-16BE":
		words = common.UTF16Decode(wordsBytes, binary.BigEndian)
	}

	// TODO - Handle wtype? Unsure what this field does, seems to always be an empty string

	prwords := ""
	for _, word := range strings.Split(words, "\t") {
		if isBadWord, _ := IsBadWord(word); isBadWord {
			prwords += "1"
		} else {
			prwords += "0"
		}
	}

	returncd := ""
	if strings.Contains(prwords, "1") {
		returncd = "040"
	} else {
		returncd = "000"
	}

	reply := map[string]string{
		"returncd": returncd,
		"prwords":  prwords,
	}

	// Only known value of this field that works this way
	if wordsRegion == "A" {
		// TODO - The real server seems to handle the input words differently per region? These values are supposed to differ from prwords
		reply["prwordsA"] = prwords
		reply["prwordsC"] = prwords
		reply["prwordsE"] = prwords
		reply["prwordsJ"] = prwords
		reply["prwordsK"] = prwords
		reply["prwordsP"] = prwords
	}

	writeAuthResponse(w, reply)
}
