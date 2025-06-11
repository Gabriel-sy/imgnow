package controller

import (
	"fmt"
	"gabrielsy/imgnow/internal/app"
	fileRepo "gabrielsy/imgnow/internal/repository/file"
	service "gabrielsy/imgnow/internal/service"
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
	fileService := service.NewFileService(fc.app)
	file, err := c.FormFile("file")
	if err != nil {
		util.LogError(err, "Failed to get file", fc.app)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	contentType := file.Header.Get("Content-Type")
	if !strings.Contains(contentType, "image/") && !strings.Contains(contentType, "video/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only image files are allowed"})
		return
	}

	urlName := c.Query("customUrl")
	customUrl, err := fileService.GenerateCustomUrl(urlName)
	if err != nil {
		util.LogError(err, "Failed to generate custom URL", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	fileRecord := &types.File{
		CustomUrl:    customUrl,
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

	// Upload async, update file status and path after upload
	go func() {
		err = fileService.UploadFile(file, customUrl)
		if err != nil {
			util.LogError(err, "Failed to upload file to R2", fc.app)
			fileRecord.Status = types.Error
			fileRepo.UpdateFileStatus(fc.app, customUrl, types.Error)
			return
		}
		fileRecord.Status = types.Active
		fileRepo.UpdateFileStatus(fc.app, customUrl, types.Active)
		fileService.UpdateFilePath(customUrl)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":   "File upload started",
		"status":    types.Pending,
		"customUrl": customUrl,
		"statusUrl": fmt.Sprintf("/api/file/status?customUrl=%s", customUrl),
	})
}

func (fc *FileController) GetFileByCustomUrl(c *gin.Context) {
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

	if file.Status == types.Pending {
		c.JSON(http.StatusTooEarly, gin.H{"error": "File is still being processed"})
		return
	}

	// Check if file has expired
	if file.ExpiresIn != nil && file.ExpiresIn.Before(time.Now()) {
		fileService := service.NewFileService(fc.app)
		err = fileService.DeleteFile(customUrl)
		if err != nil {
			util.LogError(err, "Failed to delete expired file", fc.app)
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "File has expired"})
		return
	}

	// Check if file has been deleted
	if file.DeletedAt != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File has been deleted"})
		return
	}

	// Check if file requires password
	hashedPassword, err := fileRepo.GetFilePassword(fc.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to get file password", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password"})
		return
	}

	// If file has password but no password was provided in request
	if hashedPassword != nil {
		var requestBody struct {
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":            "Password required",
				"requiresPassword": true,
			})
			return
		}

		if !util.CheckPasswordHash(requestBody.Password, *hashedPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		}
	}
	// To be used with permanent urls
	/*

		if file.Path != nil {
				c.JSON(http.StatusOK, gin.H{
					"path": *file.Path,
					})
					return
					}
	*/

	r2 := service.NewR2Service(fc.app)
	fileUrl, err := r2.GetFromR2(customUrl)
	if err != nil {
		util.LogError(err, "Failed to get file from R2", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	fileService := service.NewFileService(fc.app)

	err = fileService.TrackFileSettings(customUrl)
	if err != nil {
		util.LogError(err, "Failed to track file visualization", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track file visualization"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"path": fileUrl,
	})
}

func (fc *FileController) GetFileStatus(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{
		"status": file.Status,
	})
}

func (fc *FileController) UpdateFileSettings(c *gin.Context) {
	customUrl := c.Param("customUrl")
	if customUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Custom URL parameter is required"})
		return
	}

	var request types.FileSettings

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fileService := service.NewFileService(fc.app)
	err := fileService.HandleConfiguration(request, customUrl)
	if err != nil {
		util.LogError(err, "Failed to handle file configuration", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to handle file configuration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File settings updated successfully"})
}

func (fc *FileController) TrackVisualization(c *gin.Context) {
	customUrl := c.Param("customUrl")
	if customUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Custom URL parameter is required"})
		return
	}

	fileService := service.NewFileService(fc.app)
	err := fileService.TrackFileSettings(customUrl)
	if err != nil {
		util.LogError(err, "Failed to track file visualization", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track file visualization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Visualization tracked successfully"})
}

func (fc *FileController) CleanupExpiredFiles(c *gin.Context) {
	fileService := service.NewFileService(fc.app)
	err := fileService.CleanupExpiredFiles()
	if err != nil {
		util.LogError(err, "Failed to cleanup expired files", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup expired files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired files cleanup completed"})
}

func (fc *FileController) AddDownload(c *gin.Context) {
	customUrl := c.Param("customUrl")
	if customUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Custom URL parameter is required"})
		return
	}

	fileService := service.NewFileService(fc.app)
	err := fileService.TrackFileDownload(customUrl)
	if err != nil {
		util.LogError(err, "Failed to track file download", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track file download"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Download added successfully"})
}

func (fc *FileController) GetFileInfo(c *gin.Context) {
	customUrl := c.Param("customUrl")
	if customUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Custom URL parameter is required"})
		return
	}

	fileService := service.NewFileService(fc.app)
	file, err := fileService.GetFileInfo(customUrl)
	if err != nil {
		util.LogError(err, "Failed to get file info", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get file info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"customUrl":                  file.CustomUrl,
		"originalName":               file.OriginalName,
		"size":                       file.Size,
		"type":                       file.Type,
		"createdAt":                  file.CreatedAt,
		"status":                     file.Status,
		"path":                       file.Path,
		"expiresIn":                  file.ExpiresIn,
		"deletedAt":                  file.DeletedAt,
		"vizualizations":             file.Vizualizations,
		"downloads":                  file.Downloads,
		"deletesAfterDownload":       file.DeletesAfterDownload,
		"downloadsForDeletion":       file.DownloadsForDeletion,
		"deletesAfterVizualizations": file.DeletesAfterVizualizations,
		"vizualizationsForDeletion":  file.VizualizationsForDeletion,
	})
}
