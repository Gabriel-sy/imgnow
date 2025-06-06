package router

import (
	"gabrielsy/imgnow/internal/app"
	controller "gabrielsy/imgnow/internal/controller/file"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(app *app.Application) *gin.Engine {
	r := gin.Default()

	fileController := controller.NewFileController(app)
	r.POST("/upload", fileController.UploadFile)

	return r
}
