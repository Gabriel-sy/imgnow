package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gabrielsy/imgnow/internal/app"
	fileRepo "gabrielsy/imgnow/internal/repository/file"
	"gabrielsy/imgnow/internal/types"
	"gabrielsy/imgnow/internal/util"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nfnt/resize"
	amqp "github.com/rabbitmq/amqp091-go"
)

type FileService struct {
	app         *app.Application
	amqpConn    *amqp.Connection
	amqpChannel *amqp.Channel
}

func NewFileService(app *app.Application) *FileService {
	amqpConn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		util.LogError(err, "Failed to connect to RabbitMQ", app)
		return nil
	}
	amqpChannel, err := amqpConn.Channel()
	if err != nil {
		util.LogError(err, "Failed to open a channel", app)
		return nil
	}
	return &FileService{app: app, amqpConn: amqpConn, amqpChannel: amqpChannel}
}

func (fs *FileService) UploadToR2(file *multipart.FileHeader, customUrl string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	var body io.Reader = src
	var contentLength int64 = file.Size

	contentType := file.Header.Get("Content-Type")
	if strings.Contains(contentType, "image/") {
		body, contentLength, err = fs.handleImageCompression(file, contentType)
		if err != nil {
			util.LogError(err, "Failed to handle image compression", fs.app)
			return err
		}
	}

	if strings.Contains(contentType, "video/") {
		body, contentLength, err = fs.handleVideoCompression(file)
		if err != nil {
			util.LogError(err, "Failed to handle video compression", fs.app)
			return err
		}
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", util.GetEnv("R2_ACCOUNT_ID", fs.app)),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     util.GetEnv("R2_ACCESS_KEY_ID", fs.app),
					SecretAccessKey: util.GetEnv("R2_SECRET_ACCESS_KEY", fs.app),
				}, nil
			},
		))),
	)
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(util.GetEnv("R2_BUCKET_NAME", fs.app)),
		Key:           aws.String(customUrl),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(contentLength),
	})

	return err
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

func (fs *FileService) GetFromR2(customUrl string) (string, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", util.GetEnv("R2_ACCOUNT_ID", fs.app)),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     util.GetEnv("R2_ACCESS_KEY_ID", fs.app),
					SecretAccessKey: util.GetEnv("R2_SECRET_ACCESS_KEY", fs.app),
				}, nil
			},
		))),
	)
	if err != nil {
		return "", err
	}

	client := s3.NewFromConfig(cfg)

	presignClient := s3.NewPresignClient(client)
	presignedUrl, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(util.GetEnv("R2_BUCKET_NAME", fs.app)),
		Key:    aws.String(customUrl),
	})
	if err != nil {
		return "", err
	}

	return presignedUrl.URL, nil
}

func (fs *FileService) UpdateFilePath(customUrl string) error {
	fileUrl, err := fs.GetFromR2(customUrl)
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

func compressImage(src multipart.File, contentType string) (*bytes.Buffer, error) {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	img, _, err := image.Decode(src)
	if err != nil {
		return nil, err
	}

	maxSize := uint(1920)
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	var newWidth, newHeight uint
	if width > height {
		newWidth = maxSize
		newHeight = uint(float64(height) * float64(maxSize) / float64(width))
	} else {
		newHeight = maxSize
		newWidth = uint(float64(width) * float64(maxSize) / float64(height))
	}

	resized := resize.Resize(newWidth, newHeight, img, resize.Lanczos3)

	compressed := &bytes.Buffer{}

	if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		err = jpeg.Encode(compressed, resized, &jpeg.Options{Quality: 80})
	} else if strings.Contains(contentType, "png") {
		err = png.Encode(compressed, resized)
	} else {
		return nil, fmt.Errorf("unsupported image format: %s", contentType)
	}

	if err != nil {
		return nil, err
	}
	return compressed, nil
}

func (fs *FileService) handleImageCompression(file *multipart.FileHeader, contentType string) (io.Reader, int64, error) {
	src, err := file.Open()
	if err != nil {
		return nil, 0, err
	}
	defer src.Close()

	var body io.Reader = src
	var contentLength int64 = file.Size

	compressedBody, err := compressImage(src, contentType)
	if err != nil {
		util.LogError(err, "Could not compress image, using original", fs.app)
	} else {
		if int64(compressedBody.Len()) < file.Size {
			body = compressedBody
			contentLength = int64(compressedBody.Len())
		} else {
			if _, err := src.Seek(0, io.SeekStart); err != nil {
				return nil, 0, fmt.Errorf("failed to seek original file after failed compression: %w", err)
			}
		}
	}
	return body, contentLength, nil
}

func (fs *FileService) handleVideoCompression(file *multipart.FileHeader) (io.Reader, int64, error) {
	src, err := file.Open()
	if err != nil {
		return nil, 0, err
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %w", err)
	}

	requestID := util.GenerateHash()
	filename := requestID + filepath.Ext(file.Filename)
	message := types.VideoMessage{
		Filename:  filename,
		Content:   fileBytes,
		RequestID: requestID,
	}

	// Setup response queue with unique name
	responseQueue, err := fs.setupResponseQueue(requestID)
	if err != nil {
		return nil, 0, err
	}

	// Setup consumer
	msgs, err := fs.setupQueueConsumer(responseQueue.Name)
	if err != nil {
		return nil, 0, err
	}

	// Publish video for compression
	if err := fs.publishVideoForCompression(message); err != nil {
		return nil, 0, err
	}

	// Wait for response with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return fs.waitForCompressedVideo(ctx, msgs, requestID)
}

func (fs *FileService) setupResponseQueue(requestID string) (*amqp.Queue, error) {
	responseQueue, err := fs.amqpChannel.QueueDeclare(
		"video_queue"+requestID, // unique queue name
		true,                    // durable
		true,                    // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		util.LogError(err, "Failed to declare response queue", fs.app)
		return nil, fmt.Errorf("failed to declare response queue: %w", err)
	}
	return &responseQueue, nil
}

func (fs *FileService) setupQueueConsumer(queueName string) (<-chan amqp.Delivery, error) {
	msgs, err := fs.amqpChannel.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		util.LogError(err, "Failed to register a consumer", fs.app)
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}
	return msgs, nil
}

func (fs *FileService) publishVideoForCompression(message types.VideoMessage) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		util.LogError(err, "Failed to marshal message", fs.app)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return fs.amqpChannel.PublishWithContext(context.TODO(),
		"",
		"video_compress_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        messageBytes,
			ReplyTo:     "video_queue" + message.RequestID,
		},
	)
}

func (fs *FileService) waitForCompressedVideo(ctx context.Context, msgs <-chan amqp.Delivery, requestID string) (io.Reader, int64, error) {
	for {
		select {
		case msg := <-msgs:
			var response types.VideoMessage
			if err := json.Unmarshal(msg.Body, &response); err != nil {
				util.LogError(err, "Failed to unmarshal response", fs.app)
				return nil, 0, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			if response.RequestID == requestID {
				return bytes.NewReader(response.Content), int64(len(response.Content)), nil
			}
		case <-ctx.Done():
			util.LogError(nil, "Timeout waiting for video compression response", fs.app)
			return nil, 0, fmt.Errorf("timeout waiting for video compression response")
		}
	}
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

	if file != nil && file.DeletesAfterDownload && file.DownloadsForDeletion != nil {
		if file.Downloads >= *file.DownloadsForDeletion {
			err = fs.DeleteFile(customUrl)
			if err != nil {
				util.LogError(err, "Failed to delete file after download limit", fs.app)
				return err
			}
		}
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

func (fs *FileService) DeleteFromR2(customUrl string) error {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", util.GetEnv("R2_ACCOUNT_ID", fs.app)),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     util.GetEnv("R2_ACCESS_KEY_ID", fs.app),
					SecretAccessKey: util.GetEnv("R2_SECRET_ACCESS_KEY", fs.app),
				}, nil
			},
		))),
	)
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(util.GetEnv("R2_BUCKET_NAME", fs.app)),
		Key:    aws.String(customUrl),
	})

	return err
}

func (fs *FileService) DeleteFile(customUrl string) error {
	err := fs.DeleteFromR2(customUrl)
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
