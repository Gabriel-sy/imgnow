package router

import (
	"gabrielsy/imgnow/internal/app"
	controller "gabrielsy/imgnow/internal/controller/file"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(app *app.Application) *gin.Engine {
	r := gin.Default()

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	fileController := controller.NewFileController(app)
	r.POST("/api/file/upload", fileController.UploadFile)
	r.POST("/api/file/:customUrl", fileController.GetFileByCustomUrl)
	r.GET("/api/file/:customUrl", fileController.GetFileByCustomUrl)
	r.GET("/api/file/:customUrl/status", fileController.GetFileStatus)
	r.GET("/api/file/:customUrl/info", fileController.GetFileInfo)

	r.PUT("/api/file/:customUrl/settings", fileController.UpdateFileSettings)
	r.PUT("/api/file/:customUrl/addDownload", fileController.AddDownload)

	return r
}
