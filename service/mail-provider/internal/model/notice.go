package model

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

type GmailNotice struct {
	EmailAddress string
	HistoryID    string
	MessageID    string
}

type PulledNotice struct {
	AckID  string
	Notice GmailNotice
	Raw    string
}

type historyIDValue string

func (h *historyIDValue) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*h = ""
		return nil
	}

	if strings.HasPrefix(trimmed, `"`) {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		*h = historyIDValue(strings.TrimSpace(text))
		return nil
	}

	*h = historyIDValue(trimmed)
	return nil
}

func DecodePubSubData(encoded string) ([]byte, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, ErrInvalidPayload
	}

	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		if decoded, err := encoding.DecodeString(trimmed); err == nil {
			return decoded, nil
		}
	}
	return nil, ErrInvalidPayload
}

func ParseGmailNotice(data []byte) (GmailNotice, error) {
	var payload struct {
		EmailAddress string         `json:"emailAddress"`
		HistoryID    historyIDValue `json:"historyId"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return GmailNotice{}, ErrInvalidPayload
	}

	email := strings.ToLower(strings.TrimSpace(payload.EmailAddress))
	historyID := strings.TrimSpace(string(payload.HistoryID))
	if email == "" || historyID == "" || !isDigits(historyID) {
		return GmailNotice{}, ErrInvalidPayload
	}

	return GmailNotice{EmailAddress: email, HistoryID: historyID}, nil
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
