package app

import (
	"fmt"

	"go.uber.org/zap"
)

// initLogger создает и настраивает логгер
func initLogger(logLevel string) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if logLevel == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	return logger, nil
}
