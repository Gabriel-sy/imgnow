package util

import (
	"gabrielsy/imgnow/internal/app"
)

func LogError(err error, message string, app *app.Application) {
	if err != nil {
		app.Logger.Println(message, err)
	}
}

func LogInfo(message string, app *app.Application) {
	app.Logger.Println(message)
}
