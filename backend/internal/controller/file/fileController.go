package controller

import (
	"fmt"
	"gabrielsy/imgnow/internal/app"
	fileRepo "gabrielsy/imgnow/internal/repository/file"
	"gabrielsy/imgnow/internal/types"
	"gabrielsy/imgnow/internal/util"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type FileController struct {
	app *app.Application
}

func NewFileController(app *app.Application) *FileController {
	return &FileController{
		app: app,
	}
}

func (fc *FileController) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		util.LogError(err, "Failed to get file", fc.app)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file"})
		return
	}

	err = godotenv.Load(".env")
	if err != nil {
		util.LogError(err, "Failed to load .env file", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load .env file"})
		return
	}
	websiteName := os.Getenv("WEBSITE_URL")

	var hash string
	var exists bool
	for {
		hash = util.GenerateHash()
		exists, err = fileRepo.HashExists(fc.app, hash)
		if err != nil {
			util.LogError(err, "Failed to check hash existence", fc.app)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}
		if !exists {
			break
		}
	}

	fileRecord := &types.File{
		Hash:         hash,
		Path:         fmt.Sprintf("%s/%s", websiteName, hash),
		OriginalName: file.Filename,
		Size:         int(file.Size),
		Type:         file.Header.Get("Content-Type"),
		CreatedAt:    time.Now(),
	}

	err = fileRepo.CreateFile(fc.app, fileRecord)
	if err != nil {
		util.LogError(err, "Failed to create file record", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file information"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully", "path": fileRecord.Path})
}
