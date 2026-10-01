package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestDecodeNoticeNormalizesEmail(t *testing.T) {
	payload := `{"emailAddress":"User@Example.com","historyId":" 123 "}`
	envelope, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"data":      base64.StdEncoding.EncodeToString([]byte(payload)),
			"messageId": "m1",
		},
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	email, historyID, messageID, err := decodeNotice(envelope)
	if err != nil {
		t.Fatalf("decode notice: %v", err)
	}
	if email != "user@example.com" {
		t.Fatalf("email phải chuẩn hoá chữ thường, nhận %q", email)
	}
	if historyID != "123" {
		t.Fatalf("historyId phải được trim, nhận %q", historyID)
	}
	if messageID != "m1" {
		t.Fatalf("messageId sai: %q", messageID)
	}
}

func TestDecodeNoticeRejectsInvalidPayloads(t *testing.T) {
	cases := map[string]string{
		"không phải json": `not-json`,
		"thiếu email":     `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(`{"historyId":"1"}`)) + `"}}`,
		"thiếu historyId": `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(`{"emailAddress":"a@b.c"}`)) + `"}}`,
		"data rỗng":       `{"message":{"data":""}}`,
	}
	for name, body := range cases {
		if _, _, _, err := decodeNotice([]byte(body)); err == nil {
			t.Fatalf("trường hợp %q phải trả lỗi", name)
		}
	}
}

func TestEnvelopePayloadDecodesNotice(t *testing.T) {
	payload := `{"emailAddress":"user@example.com","historyId":123}`
	envelope, err := json.Marshal(map[string]any{
		"message": map[string]any{"data": base64.StdEncoding.EncodeToString([]byte(payload))},
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if got := envelopePayload(envelope); got != payload {
		t.Fatalf("envelopePayload sai: %q", got)
	}
}

func TestEnvelopePayloadFallsBackToRaw(t *testing.T) {
	if got := envelopePayload([]byte(`not-json`)); got != "not-json" {
		t.Fatalf("fallback phải trả body thô, nhận %q", got)
	}
	if got := envelopePayload([]byte(`{"message":{"data":"!!!"}}`)); got != "!!!" {
		t.Fatalf("base64 hỏng phải trả data thô, nhận %q", got)
	}
}

func TestNotificationVerifierSkipsWhenAudienceMissing(t *testing.T) {
	verifier := &notificationVerifier{}
	if err := verifier.verify(context.Background(), ""); err != nil {
		t.Fatalf("thiếu audience thì bỏ qua xác thực, nhận %v", err)
	}
}

func TestNotificationVerifierRequiresBearerToken(t *testing.T) {
	verifier := &notificationVerifier{audience: "https://example.com/mail"}

	if err := verifier.verify(context.Background(), ""); err == nil {
		t.Fatal("thiếu Authorization phải bị từ chối")
	}
	if err := verifier.verify(context.Background(), "Basic abc"); err == nil {
		t.Fatal("scheme không phải Bearer phải bị từ chối")
	}
}
