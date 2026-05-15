package main

import (
	"log/slog"
	"os"
)

var (
	log         *slog.Logger
	envVMURL    = os.Getenv("VICTORIA_METRICS")
	envLogLevel = os.Getenv("LOG_LEVEL")
)

func init() {
	level := slog.LevelInfo

	if envLogLevel == "debug" {
		level = slog.LevelDebug
	}

	log = slog.New(
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

	if envVMURL == "" {
		panic("No VICTORIA_METRICS env set.")
	}
}
