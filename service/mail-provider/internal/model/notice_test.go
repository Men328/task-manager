package model

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestDecodePubSubDataAcceptsAllBase64Variants(t *testing.T) {
	value := "hello pubsub"
	raw := []byte(value)
	for _, encoded := range []string{
		base64.StdEncoding.EncodeToString(raw),
		base64.RawStdEncoding.EncodeToString(raw),
		base64.URLEncoding.EncodeToString(raw),
		base64.RawURLEncoding.EncodeToString(raw),
	} {
		decoded, err := DecodePubSubData(encoded)
		if err != nil {
			t.Fatalf("decode %q: %v", encoded, err)
		}
		if string(decoded) != value {
			t.Fatalf("decode sai: %q", string(decoded))
		}
	}
}

func TestDecodePubSubDataRejectsInvalidInput(t *testing.T) {
	if _, err := DecodePubSubData("   "); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("chuỗi rỗng phải là ErrInvalidPayload, nhận %v", err)
	}
	if _, err := DecodePubSubData("!!!không-phải-base64!!!"); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("base64 hỏng phải là ErrInvalidPayload, nhận %v", err)
	}
}

func TestParseGmailNoticeNormalizesFields(t *testing.T) {
	notice, err := ParseGmailNotice([]byte(`{"emailAddress":"User@Example.com","historyId":" 123 "}`))
	if err != nil {
		t.Fatalf("parse notice: %v", err)
	}
	if notice.EmailAddress != "user@example.com" {
		t.Fatalf("email phải chuẩn hoá chữ thường, nhận %q", notice.EmailAddress)
	}
	if notice.HistoryID != "123" {
		t.Fatalf("historyId phải được trim, nhận %q", notice.HistoryID)
	}
}

func TestParseGmailNoticeAcceptsNumericHistoryID(t *testing.T) {
	notice, err := ParseGmailNotice([]byte(`{"emailAddress":"User@Example.com","historyId":1234567890}`))
	if err != nil {
		t.Fatalf("historyId dạng số phải parse được: %v", err)
	}
	if notice.HistoryID != "1234567890" {
		t.Fatalf("historyId phải chuyển thành chuỗi số, nhận %q", notice.HistoryID)
	}
}

func TestParseGmailNoticeAcceptsStringHistoryID(t *testing.T) {
	notice, err := ParseGmailNotice([]byte(`{"emailAddress":"user@example.com","historyId":"987654321"}`))
	if err != nil {
		t.Fatalf("historyId dạng chuỗi phải parse được: %v", err)
	}
	if notice.HistoryID != "987654321" {
		t.Fatalf("historyId sai: %q", notice.HistoryID)
	}
}

func TestParseGmailNoticeRejectsNonNumericHistoryID(t *testing.T) {
	if _, err := ParseGmailNotice([]byte(`{"emailAddress":"user@example.com","historyId":"abc"}`)); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("historyId không phải số phải là ErrInvalidPayload, nhận %v", err)
	}
}

func TestParseGmailNoticeRejectsInvalidPayloads(t *testing.T) {
	cases := map[string]string{
		"không phải json": `not-json`,
		"thiếu email":     `{"historyId":"1"}`,
		"thiếu historyId": `{"emailAddress":"a@b.c"}`,
		"email rỗng":      `{"emailAddress":"   ","historyId":"1"}`,
	}
	for name, payload := range cases {
		if _, err := ParseGmailNotice([]byte(payload)); !errors.Is(err, ErrInvalidPayload) {
			t.Fatalf("trường hợp %q phải là ErrInvalidPayload, nhận %v", name, err)
		}
	}
}
