package gpcm

import (
	"os"
	"strings"
)

var motdFilepath = "./motd.txt"

func GetMessageOfTheDay() (string, error) {
	contents, err := os.ReadFile(motdFilepath)
	if err != nil {
		return "", err
	}

	strContents := string(contents)
	strContents = strings.TrimSpace(strContents)

	return strContents, nil
}
