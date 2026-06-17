package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestColorHandlerWritesColoredRequestLog(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	var out bytes.Buffer
	handler := NewColorHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo})
	record := slog.NewRecord(time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC), slog.LevelInfo, "request", 0)
	record.AddAttrs(
		slog.String("method", "POST"),
		slog.String("path", "/oauth/client_token/"),
		slog.Int("status", 200),
		slog.Duration("duration", 12*time.Millisecond),
	)

	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"\033[", "INFO", "request", "method", "POST", "status", "200"} {
		if !strings.Contains(got, want) {
			t.Fatalf("log output missing %q: %q", want, got)
		}
	}
}

func TestColorHandlerHonorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var out bytes.Buffer
	handler := NewColorHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo})
	record := slog.NewRecord(time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC), slog.LevelInfo, "request", 0)
	record.AddAttrs(slog.Int("status", 500))

	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\033[") {
		t.Fatalf("expected no ANSI escapes when NO_COLOR is set: %q", out.String())
	}
}
