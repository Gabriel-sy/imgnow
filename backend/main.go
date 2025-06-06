package main

import (
	"gabrielsy/imgnow/internal/app"
	"gabrielsy/imgnow/internal/router"
	"gabrielsy/imgnow/internal/util"
	"os"
)

func main() {
	app, err := app.NewApplication()
	if err != nil {
		util.LogError(err, "Failed to create application", app)
		os.Exit(1)
	}

	r := router.SetupRoutes(app)
	r.Run(":8080")
}
