package util

import (
	"gabrielsy/imgnow/internal/app"
	"os"

	"github.com/joho/godotenv"
)

func GetEnv(key string, app *app.Application) string {
	err := godotenv.Load(".env")
	if err != nil {
		LogError(err, "Failed to load .env file", app)
		return ""
	}
	return os.Getenv(key)
}
