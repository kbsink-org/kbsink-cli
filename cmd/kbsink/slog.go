package main

import (
	"log/slog"
	"os"

	klog "github.com/kbsink-org/kbsink/pkg/logger"
)

func setupSlog(level string) error {
	min, err := klog.ParseLevel(level)
	if err != nil {
		return err
	}
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: klogSlogLevel(min)})
	slog.SetDefault(slog.New(h))
	return nil
}

func klogSlogLevel(l klog.Level) slog.Level {
	switch l {
	case klog.LevelDebug:
		return slog.LevelDebug
	case klog.LevelInfo:
		return slog.LevelInfo
	case klog.LevelWarn:
		return slog.LevelWarn
	case klog.LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
