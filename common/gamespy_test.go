package common

import (
	"fmt"
	"testing"
)

func TestParseGameSpyMessage(t *testing.T) {
	_, err := ParseGameSpyMessage(`\login\\challenge\rc5lU5V5skphHnc1eSuXh8j2EzyI2TZP\authtoken\NDSLplmebUYep9V3q2CuPf5HRWoz0K3wJNzO1XqJog0QKHTIJczIu89ecAfhwKDlngIEztYsOJoH8c4zPrp\partnerid\11\response\394beecb14fa59feb8d4c3690975e24c\firewall\1\port\0\productid\11059\gamename\mariokartwii\namespaceid\16\sdkrevision\3\quiet\0\id\1\final\`)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateGameSpyMessage(t *testing.T) {
	fmt.Println(CreateGameSpyMessage(GameSpyCommand{
		Command:      "lc",
		CommandValue: "2",
		OtherValues: map[string]string{
			"sesskey":    "07187200",
			"proof":      "b0a0e576b28861f2512b943daf374158",
			"userid":     "8467681766588",
			"profileid":  "1",
			"uniquenick": "7me4ijr5sRMCJ3cf1asa@nds",
			"lt":         "MDEyMzQ1Njc4OTBBQkNERUY=",
			"id":         "1",
		},
	}))
}
