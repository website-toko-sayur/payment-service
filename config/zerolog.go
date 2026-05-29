package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func NewLogger(appEnv string, logLevel string, serviceName string) {
	// Default log level
	level := zerolog.InfoLevel

	parsedLevel, err := zerolog.ParseLevel(
		strings.ToLower(logLevel),
	)
	if err == nil {
		level = parsedLevel
	}

	var logger zerolog.Logger

	// Development logger (pretty console)
	if appEnv == "development" {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
			NoColor:    false,
		}

		consoleWriter.FormatLevel = func(i any) string {
			level := strings.ToUpper(fmt.Sprintf("%s", i))

			switch level {
			case "DEBUG":
				return "\033[36m| DEBUG |\033[0m"
			case "INFO":
				return "\033[32m| INFO  |\033[0m"
			case "WARN":
				return "\033[33m| WARN  |\033[0m"
			case "ERROR":
				return "\033[31m| ERROR |\033[0m"
			case "FATAL":
				return "\033[35m| FATAL |\033[0m"
			default:
				return fmt.Sprintf("| %-5s|", level)
			}
		}

		consoleWriter.FormatMessage = func(i any) string {
			return fmt.Sprintf(" %s", i)
		}

		logger = zerolog.New(consoleWriter)
	} else {
		// Production logger (JSON)
		logger = zerolog.New(os.Stdout)
	}

	log.Logger = logger.
		Level(level).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	zerolog.TimeFieldFormat = time.RFC3339
}
