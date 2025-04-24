package config

import (
	"log/slog"
	"os"
)

type AppEnv string

const (
	Testing     AppEnv = "test"
	Development AppEnv = "dev"
	Production  AppEnv = "prod"
)

type AppSettings struct {
	RepositoriesLocation string
	ClonesLocation       string
	Debug                bool
	AppEnv               AppEnv
	DefaultBranch        string
}

func BaseSettings() AppSettings {
	settings := AppSettings{
		RepositoriesLocation: "repositories",
		ClonesLocation:       "clones",
		Debug:                false,
		DefaultBranch:        "main",
	}

	slog.SetLogLoggerLevel(slog.LevelInfo)

	return settings
}

func DevelopmentSettings() AppSettings {
	settings := BaseSettings()
	settings.AppEnv = Development

	slog.SetLogLoggerLevel(slog.LevelDebug)

	return settings
}

func ProductionSettings() AppSettings {
	settings := BaseSettings()
	settings.AppEnv = Production
	return settings
}

func TestSettings() AppSettings {
	settings := DevelopmentSettings()
	settings.AppEnv = Testing
	return settings
}

var Settings = func() AppSettings {
	appEnv := os.Getenv("APP_ENV")
	var config AppSettings
	switch appEnv {
	case "test":
		config = TestSettings()
	case "dev":
		config = DevelopmentSettings()
	default:
		config = ProductionSettings()
	}

	return config
}()
