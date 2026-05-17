package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/pointnoreturn/monitor/libmetric"
)

var (
	libLog *slog.Logger = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
			ReplaceAttr: func(
				groups []string,
				a slog.Attr,
			) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		}),
	)
	appLog             *slog.Logger
	victoriaMetricsUrl = os.Getenv("VICTORIA_METRICS")
	envLogLevel        = os.Getenv("LOG_LEVEL")
)

func init() {
	level := slog.LevelInfo

	if strings.EqualFold(envLogLevel, "debug") {
		level = slog.LevelDebug
	} else if strings.EqualFold(envLogLevel, "warn") {
		level = slog.LevelWarn
	} else if strings.EqualFold(envLogLevel, "error") {
		level = slog.LevelError
	}

	appLog = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(
				groups []string,
				a slog.Attr,
			) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		}),
	)

	if victoriaMetricsUrl == "" {
		panic("No VICTORIA_METRICS env set.")
	}
	libmetric.Init(victoriaMetricsUrl, libLog)
}
