package common

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"wwfc/logging"
)

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

type webhookAuthor struct {
	Name string `json:"name,omitempty"`
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
	switch v := value.(type) {
	case string:
		return "  - ``" + strings.ReplaceAll(v, "``", "` `") + "``"
	case int:
		return "  - " + strconv.Itoa(v)
	case int32:
		return "  - " + strconv.Itoa(int(v))
	case int64:
		return "  - " + strconv.FormatInt(v, 10)
	case float32:
		return "  - " + strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return "  - " + strconv.FormatFloat(v, 'f', -1, 64)
	case []any:
		var sb strings.Builder
		for i, item := range v {
			sb.WriteString(encodeWebhookValue(item))
			if i < len(v)-1 {
				sb.WriteString("\n")
			}
		}
		return sb.String()
	default:
		return "  - (unknown)"
	}
}

func (w WebhookConfig) ReportEvent(eventType string, eventData map[string]any) {
	embed := webhookEmbed{
		Title: "> **" + eventType + "**",
	}

	if w.Author != "" {
		embed.Author = webhookAuthor{Name: w.Author}
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
		panic(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		panic("received non-2xx response: " + resp.Status)
	}
	resp.Body.Close()
}

func (w WebhookConfig) RegisterEventReporting() {
	if !w.Enabled {
		return
	}
	logging.RegisterEventCallback(w.EventTypes, func(eventType string, eventData map[string]any) {
		w.ReportEvent(eventType, eventData)
	})
}
