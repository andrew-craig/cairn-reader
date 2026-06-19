package services

import (
	"log/slog"
)

// auditEvent emits a structured log entry for security-relevant auth events.
// All entries include an "event_type" field for easy filtering in log aggregators.
//
// Example filter (jq): .event_type == "login_success"
func auditEvent(eventType string, attrs ...slog.Attr) {
	allAttrs := make([]slog.Attr, 0, len(attrs)+1)
	allAttrs = append(allAttrs, slog.String("event_type", eventType))
	allAttrs = append(allAttrs, attrs...)
	slog.LogAttrs(nil, slog.LevelInfo, "auth audit", allAttrs...)
}
