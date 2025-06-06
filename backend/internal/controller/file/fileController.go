package controller

import (
	"fmt"
	"gabrielsy/imgnow/internal/app"
	fileRepo "gabrielsy/imgnow/internal/repository/file"
	"gabrielsy/imgnow/internal/types"
	"gabrielsy/imgnow/internal/util"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

	urlName := c.Query("urlName")
	var customUrl string
	var exists bool

	// urlName provided, use it as url
	if urlName != "" {
		exists, err = fileRepo.HashExists(fc.app, urlName)
		if err != nil {
			util.LogError(err, "Failed to check url name existence", fc.app)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "URL name already exists"})
			return
		}
		customUrl = urlName
	} else {
		// urlName not provided, generate a random 5 digit hash
		for {
			customUrl = util.GenerateHash()
			exists, err = fileRepo.HashExists(fc.app, customUrl)
			if err != nil {
				util.LogError(err, "Failed to check hash existence", fc.app)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
				return
			}
			if !exists {
				break
			}
		}
	}

	websiteName := util.GetEnv("WEBSITE_URL", fc.app)
	fileRecord := &types.File{
		CustomUrl:    customUrl,
		Path:         fmt.Sprintf("%s/%s", websiteName, customUrl),
		OriginalName: strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)),
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

func (fc *FileController) GetFileByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hash parameter is required"})
		return
	}

	file, err := fileRepo.FindHash(fc.app, hash)
	if err != nil {
		util.LogError(err, "Failed to find file", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path": file.Path,
	})
}
