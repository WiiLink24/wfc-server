package logging

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/logrusorgru/aurora/v3"
)

var (
	logDir   = "./logs"
	logLevel = 0
)

func SetLevel(level int) {
	logLevel = level
}

func SetOutput(output string) error {
	switch output {
	case "None":
		log.SetOutput(io.Discard)
	case "StdOut":
		log.SetOutput(os.Stdout)
	case "StdOutAndFile":
		if err := os.MkdirAll(logDir, 0700); err != nil {
			return err
		}

		time := time.Now()
		logFilePath := time.Format(logDir + "/2006-01-02-15-04-05.log")

		file, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE, 0400)
		if err != nil {
			return err
		}

		log.SetOutput(io.MultiWriter(os.Stdout, file))
	default:
		return errors.New("invalid output value provided")
	}

	return nil
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
