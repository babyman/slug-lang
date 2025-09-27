package kernel

import (
	"os"
	"slug/internal/logger"
)

func SystemLogLevel() logger.Level {
	if envLevel := os.Getenv("KERNEL_LOG_LEVEL"); envLevel != "" {
		return logger.ParseLevel(envLevel)
	}
	return logger.ERROR
}
