package gpcm

import (
	"errors"
	"os"
)

var motdFilepath = "./motd.txt"
var motd string = ""

func GetMessageOfTheDay() (string, error) {
	if motd == "" {
		contents, err := os.ReadFile(motdFilepath)
		if err != nil {
			return "", err
		}

		motd = string(contents)
	}

	return motd, nil
}

func SetMessageOfTheDay(nmotd string) error {
	if nmotd == "" {
		return errors.New("Motd cannot be empty")
	}

	err := os.WriteFile(motdFilepath, []byte(nmotd), 0644)
	motd = nmotd

	return err
}
