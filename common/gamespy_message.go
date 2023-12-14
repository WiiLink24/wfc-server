package common

import (
	"errors"
	"fmt"
	"strings"
)

type GameSpyCommand struct {
	Command      string
	CommandValue string
	OtherValues  map[string]string
}

var (
	InvalidGameSpyCommand = errors.New("invalid GameSpy command received")
)

func ParseGameSpyMessage(msg string) ([]GameSpyCommand, error) {
	if !strings.Contains(msg, `\final\`) {
		return nil, InvalidGameSpyCommand
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
				return nil, InvalidGameSpyCommand
			}

			key := msg[1:keyEnd]
			value := ""
			msg = msg[keyEnd+1:]

			if key == "final" {
				// We are done.
				break
			}

			if strings.Contains(msg, `\`) {
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

func CreateGameSpyMessage(command GameSpyCommand) string {
	query := ""
	for k, v := range command.OtherValues {
		query += fmt.Sprintf(`\%s\%s`, strings.Replace(k, `\`, ``, -1), strings.Replace(v, `\`, ``, -1))
	}

	if command.Command != "" {
		query = fmt.Sprintf(`\%s\%s%s`, command.Command, command.CommandValue, query)
	}

	return query + `\final\`
}
