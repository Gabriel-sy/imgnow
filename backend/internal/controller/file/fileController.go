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

	urlName := c.Query("urlName")
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

	fileService := service.NewFileService(fc.app)

	if file.ExpiresIn != nil && file.ExpiresIn.Before(time.Now()) {
		err = fileService.DeleteFile(customUrl)
		if err != nil {
			util.LogError(err, "Failed to delete expired file", fc.app)
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "File has expired"})
		return
	}

	if file.DeletedAt != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File has been deleted"})
		return
	}

	err = fileService.TrackFileSettings(customUrl)
	if err != nil {
		util.LogError(err, "Failed to track file visualization", fc.app)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track file visualization"})
		return
	}

	if file.Path != nil {
		c.JSON(http.StatusOK, gin.H{
			"path": *file.Path,
		})
		return
	}

	// Used as a fallback only
	util.LogError(nil, fmt.Sprintf("Getting file from R2 as fallback, something went very wrong %s", customUrl), fc.app)
	r2 := service.NewR2Service(fc.app)
	fileUrl, err := r2.GetFromR2(customUrl)
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
	})
}

func (fc *FileController) UpdateFileSettings(c *gin.Context) {
	customUrl := c.Param("customUrl")
	if customUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Custom URL parameter is required"})
		return
	}

	var request struct {
		ExpiresIn                  *time.Time `json:"expiresIn"`
		DeletesAfterDownload       bool       `json:"deletesAfterDownload"`
		DownloadsForDeletion       *int       `json:"downloadsForDeletion"`
		DeletesAfterVizualizations bool       `json:"deletesAfterVizualizations"`
		VizualizationsForDeletion  *int       `json:"vizualizationsForDeletion"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fileService := service.NewFileService(fc.app)

	// Update expiration if provided
	if request.ExpiresIn != nil {
		err := fileService.UpdateFileExpiration(customUrl, request.ExpiresIn)
		if err != nil {
			util.LogError(err, "Failed to update file expiration", fc.app)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update file expiration"})
			return
		}
	}

	// Update deletion settings if any are provided
	if request.DeletesAfterDownload || request.DeletesAfterVizualizations {
		err := fileService.UpdateDeletionSettings(
			customUrl,
			request.DeletesAfterDownload,
			request.DownloadsForDeletion,
			request.DeletesAfterVizualizations,
			request.VizualizationsForDeletion,
		)
		if err != nil {
			util.LogError(err, "Failed to update deletion settings", fc.app)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update deletion settings"})
			return
		}
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
