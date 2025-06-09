package router

import (
	"gabrielsy/imgnow/internal/app"
	controller "gabrielsy/imgnow/internal/controller/file"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(app *app.Application) *gin.Engine {
	r := gin.Default()

	fileController := controller.NewFileController(app)
	r.POST("/api/file/upload", fileController.UploadFile)

	r.GET("/api/file/:customUrl/status", fileController.GetFileStatus)
	r.GET("/api/file/:customUrl", fileController.GetFileByCustomUrl)
	r.GET("/api/file/:customUrl/info", fileController.GetFileInfo)

	r.PUT("/api/file/:customUrl/settings", fileController.UpdateFileSettings)
	r.PUT("/api/file/:customUrl/addDownload", fileController.AddDownload)

	return r
}
