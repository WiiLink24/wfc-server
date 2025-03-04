package gpcm

import (
	"os"
)

var motdFilepath = "./motd.txt"

func GetMessageOfTheDay() (string, error) {
	contents, err := os.ReadFile(motdFilepath)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}

