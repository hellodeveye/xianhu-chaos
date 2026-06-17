package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
)

type ColorHandler struct {
	out    io.Writer
	level  slog.Leveler
	mu     *sync.Mutex
	attrs  []slog.Attr
	groups []string
	color  bool
}

func NewColorHandler(out io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	level := slog.LevelInfo
	if opts != nil && opts.Level != nil {
		level = opts.Level.Level()
	}
	return &ColorHandler{
		out:   out,
		level: level,
		mu:    &sync.Mutex{},
		color: os.Getenv("NO_COLOR") == "",
	}
}

func (h *ColorHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *ColorHandler) Handle(_ context.Context, record slog.Record) error {
	var b strings.Builder
	timestamp := record.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	b.WriteString(colorize(timestamp.Format("15:04:05.000"), dim, h.color))
	b.WriteByte(' ')
	b.WriteString(colorize(strings.ToUpper(record.Level.String()), levelColor(record.Level), h.color))
	b.WriteByte(' ')
	b.WriteString(colorize(record.Message, bold, h.color))

	for _, attr := range h.attrs {
		h.appendAttr(&b, attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		h.appendAttr(&b, attr)
		return true
	})
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = cloneAttrs(h.attrs)
	next.attrs = append(next.attrs, attrs...)
	return &next
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.groups = cloneStrings(h.groups)
	next.groups = append(next.groups, name)
	return &next
}

func (h *ColorHandler) appendAttr(b *strings.Builder, attr slog.Attr) {
	if attr.Key == "" {
		return
	}
	attr.Value = attr.Value.Resolve()
	b.WriteByte(' ')
	b.WriteString(colorize(h.attrKey(attr.Key), dim, h.color))
	b.WriteByte('=')
	b.WriteString(h.attrValue(attr))
}

func (h *ColorHandler) attrKey(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	return strings.Join(append(cloneStrings(h.groups), key), ".")
}

func (h *ColorHandler) attrValue(attr slog.Attr) string {
	value := slogValueString(attr.Value)
	switch attr.Key {
	case "method":
		return colorize(value, methodColor(value), h.color)
	case "status":
		return colorize(value, statusColor(value), h.color)
	case "duration":
		return colorize(value, magenta, h.color)
	case "error":
		return colorize(value, red, h.color)
	case "addr", "path":
		return colorize(value, cyan, h.color)
	default:
		return value
	}
}

func slogValueString(value slog.Value) string {
	switch value.Kind() {
	case slog.KindString:
		return quoteIfNeeded(value.String())
	case slog.KindInt64:
		return strconv.FormatInt(value.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(value.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(value.Float64(), 'f', -1, 64)
	case slog.KindBool:
		return strconv.FormatBool(value.Bool())
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339)
	default:
		return quoteIfNeeded(fmt.Sprint(value.Any()))
	}
}

func quoteIfNeeded(value string) string {
	if value == "" {
		return `""`
	}
	if strings.ContainsAny(value, " \t\n\r\"=") {
		return strconv.Quote(value)
	}
	return value
}

func levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return red
	case level >= slog.LevelWarn:
		return yellow
	case level >= slog.LevelInfo:
		return green
	default:
		return white
	}
}

func methodColor(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return blue
	case "POST":
		return green
	case "PUT", "PATCH":
		return yellow
	case "DELETE":
		return red
	default:
		return cyan
	}
}

func statusColor(status string) string {
	if status == "" {
		return white
	}
	switch status[0] {
	case '2':
		return green
	case '3':
		return cyan
	case '4':
		return yellow
	case '5':
		return red
	default:
		return white
	}
}

func colorize(value, color string, enabled bool) string {
	if !enabled || color == "" {
		return value
	}
	return color + value + reset
}

func cloneAttrs(in []slog.Attr) []slog.Attr {
	if len(in) == 0 {
		return nil
	}
	out := make([]slog.Attr, len(in))
	copy(out, in)
	return out
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
