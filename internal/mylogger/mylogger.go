package mylogger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

const (
	LevelDebug string = "DEBUG"
	LevelInfo  string = "INFO"
	LevelWarn  string = "WARN"
	LevelError string = "ERROR"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, err error, args ...any)
	Action(action string) Logger
	With(args ...any) Logger
	WithGroup(groupName string) Logger
}

func New(logLevel string) (Logger, error) {
	// Generate a deterministic request ID (sequential)
	requestID, err := generateRequestID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate request ID: %v", err)
	}

	// Retrieve hostname
	hostman, err := os.Hostname()
	if err != nil {
		hostman = "localhost"
	}
	level := new(slog.LevelVar)
	switch logLevel {
	case LevelDebug:
		level.Set(slog.LevelDebug)
	case LevelInfo:
		level.Set(slog.LevelInfo)
	case LevelWarn:
		level.Set(slog.LevelWarn)
	case LevelError:
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}

	// Ensure the directory exists
	if err = os.MkdirAll("logs", 0o755); err != nil {
		return nil, fmt.Errorf("cannot create logs director: %s", err)
	}
	// Ensure the logfile exists
	// logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	// if err != nil {
	// 	return nil, fmt.Errorf("cannot create app.log: %s", err)
	// }

	// multiWriter := io.MultiWriter(os.Stdout, logFile)
	multiWriter := io.MultiWriter(os.Stdout)

	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename 'msg' to 'message'
			if a.Key == slog.MessageKey {
				return slog.Attr{Key: "message", Value: a.Value}
			}
			// Format time as ISO 8601
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.Attr{Key: "timestamp", Value: slog.StringValue(t.Format(time.RFC3339))}
				}
			}
			return a
		},
	})

	log := slog.New(handler).With("hostname", hostman, "request_id", requestID)
	return &logger{
		log: log,
	}, nil
}

type logger struct {
	log *slog.Logger
}

func (l *logger) Debug(msg string, args ...any) {
	l.log.Debug(msg, args...)
}

func (l *logger) Info(msg string, args ...any) {
	l.log.Info(msg, args...)
}

func (l *logger) Warn(msg string, args ...any) {
	l.log.Warn(msg, args...)
}

// Error log with stack trace and request ID
func (l *logger) Error(msg string, err error, args ...any) {
	// Capture stack frames
	frames := captureFrames(5, 8)

	// Build the structured attributes
	attrs := append(args, slog.Group("error",
		slog.Any("msg", err),
		slog.Any("stack", frames),
	))

	// Log the error
	l.log.Error(msg, attrs...)
}

func (l logger) Action(action string) Logger {
	l.log = l.log.With("action", action)
	return &l
}

func (l logger) With(args ...any) Logger {
	l.log = l.log.With(args...)
	return &l
}

func (l logger) WithGroup(groupName string) Logger {
	l.log = l.log.WithGroup(groupName)
	return &l
}
