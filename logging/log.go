package logging

import (
	"github.com/logrusorgru/aurora/v3"
	"log"
)

func Notice(module string, arguments ...string) {
	var finalStr string
	for _, argument := range arguments {
		finalStr += argument
		finalStr += " "
	}

	log.Printf("[%s]: %s", aurora.Green(module), finalStr)
}
