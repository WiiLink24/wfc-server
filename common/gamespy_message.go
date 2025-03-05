package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type GameSpyCommand struct {
	Command      string
	CommandValue string
	OtherValues  map[string]string
}

var (
	ErrInvalidGameSpyCommand = errors.New("invalid GameSpy command received")
	ErrNoGameStatsDataLength = errors.New("no data length found in GameStats message")
)

func parseGameSpyMessage(msg string, gameStats bool) ([]GameSpyCommand, error) {
	if !strings.Contains(msg, `\final\`) {
		return nil, ErrInvalidGameSpyCommand
	}

	var commands []GameSpyCommand
	for len(msg) > 0 && string(msg[0]) == `\` && strings.Contains(msg, `\final\`) {
		foundCommand := false
		g := GameSpyCommand{
			OtherValues: map[string]string{},
		}

		for len(msg) > 0 && string(msg[0]) == `\` {
			keyEnd := strings.Index(msg[1:], `\`) + 1
			if keyEnd < 2 {
				return nil, ErrInvalidGameSpyCommand
			}

			key := msg[1:keyEnd]
			value := ""
			msg = msg[keyEnd+1:]

			if key == "final" {
				// We are done.
				break
			}

			if gameStats && key == "data" {
				if g.OtherValues["length"] == "" {
					return nil, ErrNoGameStatsDataLength
				}

				dataLength, err := strconv.Atoi(g.OtherValues["length"])
				if err != nil {
					return nil, err
				}

				if len(msg) < dataLength+1 {
					return nil, ErrInvalidGameSpyCommand
				}

				value = msg[:dataLength]
				msg = msg[dataLength:]
				if msg[0] == '\\' {
					msg = msg[1:]
				}
			} else if strings.Contains(msg, `\`) {
				if msg[0] != '\\' {
					valueEnd := strings.Index(msg[1:], `\`)
					value = msg[:valueEnd+1]
					msg = msg[valueEnd+1:]
				}
			} else {
				// We have most likely reached the end of the line.
				// However, we do not want to exit out without parsing the final key.
				value = msg
			}

			if !foundCommand {
				g.Command = key
				g.CommandValue = value
				foundCommand = true
			} else {
				g.OtherValues[key] = value
			}
		}

		commands = append(commands, g)
	}

	return commands, nil
}

func ParseGameSpyMessage(msg string) ([]GameSpyCommand, error) {
	return parseGameSpyMessage(msg, false)
}

func ParseGameStatsMessage(msg string) ([]GameSpyCommand, error) {
	return parseGameSpyMessage(msg, true)
}

func CreateGameSpyMessage(command GameSpyCommand) string {
	query := ""
	endQuery := ""
	for k, v := range command.OtherValues {
		if command.Command == "getpdr" && k == "data" {
			endQuery += fmt.Sprintf(`\%s\%s`, k, v)
		} else {
			query += fmt.Sprintf(`\%s\%s`, k, v)
		}
	}

	query += endQuery

	if command.Command != "" {
		query = fmt.Sprintf(`\%s\%s%s`, command.Command, command.CommandValue, query)
	}

	return query + `\final\`
}
