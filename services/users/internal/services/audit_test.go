package services

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestAuditEvent_EmitsStructuredLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	auditEvent("login_success",
		slog.String("user_id", "test-user-123"),
		slog.String("ip", "192.168.1.1"),
	)

	var entry map[string]interface{}
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v\nraw: %s", err, buf.String())
	}

	if got := entry["event_type"]; got != "login_success" {
		t.Errorf("event_type = %q, want %q", got, "login_success")
	}
	if got := entry["user_id"]; got != "test-user-123" {
		t.Errorf("user_id = %q, want %q", got, "test-user-123")
	}
	if got := entry["ip"]; got != "192.168.1.1" {
		t.Errorf("ip = %q, want %q", got, "192.168.1.1")
	}
	if entry["msg"] != "auth audit" {
		t.Errorf("msg = %q, want %q", entry["msg"], "auth audit")
	}
}

func TestAuditEvent_EventTypes(t *testing.T) {
	eventTypes := []string{
		"login_success",
		"login_failure",
		"account_created",
		"token_refreshed",
		"token_reuse_detected",
		"password_changed",
	}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
			slog.SetDefault(logger)

			// Should not panic
			auditEvent(eventType, slog.String("user_id", "u1"))

			var entry map[string]interface{}
			if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
				t.Fatalf("failed to parse log output: %v", err)
			}
			if got := entry["event_type"]; got != eventType {
				t.Errorf("event_type = %q, want %q", got, eventType)
			}
		})
	}
}
