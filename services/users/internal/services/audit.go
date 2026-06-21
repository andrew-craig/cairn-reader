package services

import (
	"context"
	"log/slog"
)

func auditEvent(eventType string, attrs ...slog.Attr) {
	allAttrs := make([]slog.Attr, 0, len(attrs)+1)
	allAttrs = append(allAttrs, slog.String("event_type", eventType))
	allAttrs = append(allAttrs, attrs...)
	slog.LogAttrs(context.Background(), slog.LevelInfo, "auth audit", allAttrs...)
}
