package logging

import (
	"fmt"
	"log"

	"github.com/logrusorgru/aurora/v3"
)

var logLevel = 0

func SetLevel(level int) {
	logLevel = level
}

func Notice(module string, arguments ...any) {
	if logLevel < 1 {
		return
	}

	var finalStr string
	for _, argument := range arguments {
		finalStr += fmt.Sprint(argument)
		finalStr += " "
	}

	log.Printf(aurora.BrightGreen("N[%s]").String()+": %s", module, finalStr)
}

func Error(module string, arguments ...any) {
	if logLevel < 2 {
		return
	}

	var finalStr string
	for _, argument := range arguments {
		finalStr += fmt.Sprint(argument)
		finalStr += " "
	}

	log.Printf(aurora.BrightRed("E[%s]").String()+": %s", module, finalStr)
}

func Warn(module string, arguments ...any) {
	if logLevel < 3 {
		return
	}

	var finalStr string
	for _, argument := range arguments {
		finalStr += fmt.Sprint(argument)
		finalStr += " "
	}

	log.Printf(aurora.BrightYellow("W[%s]").String()+": %s", module, finalStr)
}

func Info(module string, arguments ...any) {
	if logLevel < 4 {
		return
	}

	var finalStr string
	for _, argument := range arguments {
		finalStr += fmt.Sprint(argument)
		finalStr += " "
	}

	log.Printf(aurora.BrightCyan("I[%s]").String()+": %s", module, finalStr)
}
