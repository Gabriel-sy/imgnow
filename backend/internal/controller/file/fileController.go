package controller

import (
	"fmt"
	"gabrielsy/imgnow/internal/app"
	fileRepo "gabrielsy/imgnow/internal/repository/file"
	fileService "gabrielsy/imgnow/internal/service/file"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	contentType := file.Header.Get("Content-Type")
	if !strings.Contains(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only image files are allowed"})
		return
	}

	urlName := c.Query("urlName")
	fileService := fileService.NewFileService(fc.app)
	customUrl, err := fileService.GenerateCustomUrl(urlName)
	if err != nil {
		util.LogError(err, "Failed to generate custom URL", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	websiteName := util.GetEnv("WEBSITE_URL", fc.app)
	fileRecord := &types.File{
		CustomUrl:    customUrl,
		Path:         fmt.Sprintf("%s/%s", websiteName, customUrl),
		OriginalName: strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)),
		Size:         int(file.Size),
		Type:         file.Header.Get("Content-Type"),
		CreatedAt:    time.Now(),
		Status:       types.Pending,
	}

	err = fileRepo.CreateFile(fc.app, fileRecord)
	if err != nil {
		util.LogError(err, "Failed to create initial file record", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	go func() {
		err = fileService.UploadToR2(file, customUrl)
		if err != nil {
			util.LogError(err, "Failed to upload file to R2", fc.app)
			fileRecord.Status = types.Error
			fileRepo.UpdateFileStatus(fc.app, customUrl, types.Error)
			return
		}
		fileRecord.Status = types.Active
		fileRepo.UpdateFileStatus(fc.app, customUrl, types.Active)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":   "File upload started",
		"path":      fileRecord.Path,
		"status":    types.Pending,
		"statusUrl": fmt.Sprintf("/api/file/status?customUrl=%s", customUrl),
	})
}

func (fc *FileController) GetFileByHash(c *gin.Context) {
	customUrl := c.Param("customUrl")
	if customUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Custom URL parameter is required"})
		return
	}

	file, err := fileRepo.FindFileByCustomUrl(fc.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to find file", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	fileService := fileService.NewFileService(fc.app)
	fileUrl, err := fileService.GetFromR2(customUrl)
	if err != nil {
		util.LogError(err, "Failed to get file from R2", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path": fileUrl,
	})
}

func (fc *FileController) GetFileStatus(c *gin.Context) {
	customUrl := c.Query("customUrl")
	if customUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Custom URL parameter is required"})
		return
	}

	file, err := fileRepo.FindFileByCustomUrl(fc.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to find file", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": file.Status,
		"path":   file.Path,
	})
}
