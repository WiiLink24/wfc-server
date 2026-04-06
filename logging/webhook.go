package logging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

type WebhookAuthorConfig struct {
	Name string `xml:",chardata"`
	URL  string `xml:"url,attr,omitempty"`
}

type WebhookConfig struct {
	Enabled bool   `xml:"enabled"`
	URL     string `xml:"url"`
	// e.g. <author url="https://example.com">Author Name</author>
	Author     WebhookAuthorConfig `xml:"author,omitempty"`
	EventTypes []string            `xml:"eventTypes>event"`
}

type webhookAuthor struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type webhookEmbed struct {
	Author      webhookAuthor `json:"author,omitempty"`
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
}

type webhookPayload struct {
	Embeds []webhookEmbed `json:"embeds"`
}

func encodeWebhookValue(value any) string {
	if reflect.TypeOf(value).Kind() == reflect.Array {
		var sb strings.Builder
		s := reflect.ValueOf(value)
		for i := 0; i < s.Len(); i++ {
			sb.WriteString(encodeWebhookValue(s.Index(i).Interface()))
			if i < s.Len()-1 {
				sb.WriteString("\n")
			}
		}
		return sb.String()
	}

	return "  - ``" + strings.ReplaceAll(fmt.Sprintf("%v", value), "``", "` `") + "``"
}

func (w WebhookConfig) ReportEvent(eventType string, eventData map[string]any) {
	embed := webhookEmbed{
		Title: "> **" + eventType + "**",
	}

	if w.Author.Name != "" {
		embed.Author = webhookAuthor{Name: w.Author.Name, URL: w.Author.URL}
	}

	for key, value := range eventData {
		embed.Description += "- " + key + "\n" + encodeWebhookValue(value) + "\n"
	}

	// Send HTTP POST request
	jsonData, err := json.Marshal(webhookPayload{Embeds: []webhookEmbed{embed}})
	if err != nil {
		panic(err)
	}
	resp, err := http.Post(w.URL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		Error("LOGGING", "Failed to send webhook request:", err)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		Error("LOGGING", "Received non-2xx response from webhook:", resp.Status)
	}
	if err := resp.Body.Close(); err != nil {
		Error("LOGGING", "Failed to close webhook response body:", err)
	}

}

func (w WebhookConfig) RegisterWebhook() {
	if !w.Enabled {
		return
	}
	RegisterEventCallback(w.EventTypes, func(eventType string, eventData map[string]any) {
		w.ReportEvent(eventType, eventData)
	})
}
