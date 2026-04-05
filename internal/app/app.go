package app

import (
	"os"
	"runtime"
	"time"

	"github.com/rs/zerolog"
)

var Version string

var Logger zerolog.Logger

var Info = map[string]any{}

var StartTime = time.Now()

// DB is the shared SQLite database path
var DB string

func Init() {
	initLogger()

	Info["version"] = Version
	Info["platform"] = runtime.GOARCH

	Logger.Info().Str("version", Version).Str("platform", runtime.GOARCH).Msg("[app] start")

	DB = Env("STRIX_DB_PATH", "cameras.db")
}

func Env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
