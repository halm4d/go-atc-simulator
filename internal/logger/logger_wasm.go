//go:build js

package logger

import (
	"log/slog"
	"os"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	})))
}

func Debug(msg string, args ...any) {}
func Info(msg string, args ...any)  {}
func Warn(msg string, args ...any)  {}
func Error(msg string, args ...any) { slog.Error(msg, args...) }
