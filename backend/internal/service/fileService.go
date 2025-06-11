package service

import (
	"fmt"
	"gabrielsy/imgnow/internal/app"
	fileRepo "gabrielsy/imgnow/internal/repository/file"
	"gabrielsy/imgnow/internal/types"
	"gabrielsy/imgnow/internal/util"
	"io"
	"mime/multipart"
	"strings"
	"time"
)

type FileService struct {
	app *app.Application
}

func NewFileService(app *app.Application) *FileService {
	return &FileService{app: app}
}

func (fs *FileService) GetFileInfo(customUrl string) (*types.File, error) {
	file, err := fileRepo.FindFileByCustomUrl(fs.app, customUrl)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (fs *FileService) UploadFile(file *multipart.FileHeader, customUrl string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	var body io.Reader = src
	var contentLength int64 = file.Size

	contentType := file.Header.Get("Content-Type")
	if strings.Contains(contentType, "image/") {
		is := NewImageService(fs.app)
		body, contentLength, err = is.HandleImageCompression(file, contentType)
		if err != nil {
			util.LogError(err, "Failed to handle image compression", fs.app)
			return err
		}
	}

	if strings.Contains(contentType, "video/") {
		vs := NewVideoService(fs.app)
		body, contentLength, err = vs.HandleVideoCompression(file)
		if err != nil {
			util.LogError(err, "Failed to handle video compression", vs.app)
			return err
		}
	}

	r2 := NewR2Service(fs.app)
	err = r2.UploadToR2(body, contentType, contentLength, customUrl)
	if err != nil {
		util.LogError(err, "Failed to upload file to R2", fs.app)
		return err
	}

	return nil
}

func (fs *FileService) TrackFileDownload(customUrl string) error {
	err := fileRepo.IncrementDownloads(fs.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to track file download", fs.app)
		return err
	}

	// Check if file should be deleted based on download count
	file, err := fileRepo.GetFileDeletionInfo(fs.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to get file deletion info", fs.app)
		return err
	}

	if file != nil && file.DeletesAfterDownload && file.DownloadsForDeletion != nil {
		if file.Downloads >= *file.DownloadsForDeletion {
			err = fs.DeleteFile(customUrl)
			if err != nil {
				util.LogError(err, "Failed to delete file after download limit", fs.app)
				return err
			}
		}
	}

	return nil
}

func (fs *FileService) UpdateFilePath(customUrl string) error {
	r2 := NewR2Service(fs.app)
	fileUrl, err := r2.GetFromR2(customUrl)
	if err != nil {
		util.LogError(err, "Failed to get file from R2", fs.app)
		return err
	}
	err = fileRepo.UpdateFilePath(fs.app, customUrl, fileUrl)
	if err != nil {
		util.LogError(err, "Failed to update file path", fs.app)
		return err
	}
	return nil
}

func (fs *FileService) GenerateCustomUrl(urlName string) (string, error) {
	var customUrl string

	// urlName provided, use it as url
	if urlName != "" {
		exists, err := fileRepo.CustomUrlExists(fs.app, urlName)
		if err != nil {
			util.LogError(err, "Failed to check url name existence", fs.app)
			return "", fmt.Errorf("failed to check url name existence: %w", err)
		}
		if exists {
			return "", fmt.Errorf("url name already exists")
		}
		customUrl = urlName
	} else {
		// urlName not provided, generate a random 5 digit hash
		for {
			customUrl = util.GenerateHash()
			exists, err := fileRepo.CustomUrlExists(fs.app, customUrl)
			if err != nil {
				util.LogError(err, "Failed to check hash existence", fs.app)
				return "", fmt.Errorf("failed to check hash existence: %w", err)
			}
			if !exists {
				break
			}
		}
	}

	return customUrl, nil
}

func (fs *FileService) UpdateFileExpiration(customUrl string, expiresIn *time.Time) error {
	err := fileRepo.UpdateExpirationSettings(fs.app, customUrl, expiresIn)
	if err != nil {
		util.LogError(err, "Failed to update file expiration", fs.app)
		return err
	}
	return nil
}

func (fs *FileService) UpdateDeletionSettings(customUrl string, deletesAfterDownload bool, downloadsForDeletion *int, deletesAfterVizualizations bool, vizualizationsForDeletion *int) error {
	err := fileRepo.UpdateDeletionDownloadSettings(fs.app, customUrl, deletesAfterDownload, downloadsForDeletion)
	if err != nil {
		util.LogError(err, "Failed to update download deletion settings", fs.app)
		return err
	}

	err = fileRepo.UpdateDeletionVizualizationSettings(fs.app, customUrl, deletesAfterVizualizations, vizualizationsForDeletion)
	if err != nil {
		util.LogError(err, "Failed to update visualization deletion settings", fs.app)
		return err
	}

	return nil
}

// Returns true if file is available, false if it's not
func (fs *FileService) TrackFileSettings(customUrl string) error {
	err := fileRepo.IncrementVizualizations(fs.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to track file visualization", fs.app)
		return err
	}

	// Check if file should be deleted based on visualization count
	file, err := fileRepo.GetFileDeletionInfo(fs.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to get file deletion info", fs.app)
		return err
	}

	if file != nil && file.DeletesAfterVizualizations && file.VizualizationsForDeletion != nil {
		if file.Vizualizations >= *file.VizualizationsForDeletion {
			err = fs.DeleteFile(customUrl)
			if err != nil {
				util.LogError(err, "Failed to delete file after visualization limit", fs.app)
				return err
			}
		}
	}

	return nil
}

func (fs *FileService) CleanupExpiredFiles() error {
	expiredFiles, err := fileRepo.GetExpiredFiles(fs.app)
	if err != nil {
		util.LogError(err, "Failed to get expired files", fs.app)
		return err
	}

	for _, file := range expiredFiles {
		err = fs.DeleteFile(file.CustomUrl)
		if err != nil {
			util.LogError(err, "Failed to delete expired file", fs.app)
			continue
		}
	}

	return nil
}

func (fs *FileService) DeleteFile(customUrl string) error {
	r2 := NewR2Service(fs.app)
	err := r2.DeleteFromR2(customUrl)
	if err != nil {
		util.LogError(err, "Failed to delete file from R2", fs.app)
	}

	err = fileRepo.MarkFileAsDeleted(fs.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to mark file as deleted", fs.app)
		return err
	}

	return nil
}

func (fs *FileService) HandleConfiguration(request types.FileSettings, customUrl string) error {
	// Update expiration if provided
	if request.ExpiresIn != nil {
		err := fs.UpdateFileExpiration(customUrl, request.ExpiresIn)
		if err != nil {
			util.LogError(err, "Failed to update file expiration", fs.app)
			return err
		}
	}

	// Update deletion settings if any are provided
	if request.DeletesAfterDownload || request.DeletesAfterVizualizations {
		err := fs.UpdateDeletionSettings(
			customUrl,
			request.DeletesAfterDownload,
			request.DownloadsForDeletion,
			request.DeletesAfterVizualizations,
			request.VizualizationsForDeletion,
		)
		if err != nil {
			util.LogError(err, "Failed to update deletion settings", fs.app)
			return err
		}
	}

	// Update password if provided
	if request.Password != nil {
		err := fs.UpdatePassword(customUrl, request.Password)
		if err != nil {
			util.LogError(err, "Failed to update password", fs.app)
			return err
		}
	}

	return nil
}

func (fs *FileService) UpdatePassword(customUrl string, password *string) error {
	err := fileRepo.UpdatePassword(fs.app, customUrl, password)
	if err != nil {
		util.LogError(err, "Failed to update password", fs.app)
		return err
	}
	return nil
}

func (fs *FileService) VerifyPassword(customUrl string, password string) (bool, error) {
	hashedPassword, err := fileRepo.GetFilePassword(fs.app, customUrl)
	if err != nil {
		util.LogError(err, "Failed to get file password", fs.app)
		return false, err
	}
	return util.CheckPasswordHash(password, *hashedPassword), nil
}