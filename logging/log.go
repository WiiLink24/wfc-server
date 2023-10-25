package logging

import (
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"log"
)

func Notice(module string, arguments ...any) {
	var finalStr string
	for _, argument := range arguments {
		finalStr += fmt.Sprint(argument)
		finalStr += " "
	}

	log.Printf(aurora.BrightGreen("N[%s]").String()+": %s", module, finalStr)
}

func Error(module string, arguments ...any) {
	var finalStr string
	for _, argument := range arguments {
		finalStr += fmt.Sprint(argument)
		finalStr += " "
	}

	log.Printf(aurora.BrightRed("E[%s]").String()+": %s", module, finalStr)
}

func Warn(module string, arguments ...any) {
	var finalStr string
	for _, argument := range arguments {
		finalStr += fmt.Sprint(argument)
		finalStr += " "
	}

	log.Printf(aurora.BrightYellow("W[%s]").String()+": %s", module, finalStr)
}

func Info(module string, arguments ...any) {
	var finalStr string
	for _, argument := range arguments {
		finalStr += fmt.Sprint(argument)
		finalStr += " "
	}

	log.Printf(aurora.BrightCyan("I[%s]").String()+": %s", module, finalStr)
}
