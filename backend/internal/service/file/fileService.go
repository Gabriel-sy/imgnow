package service

import (
	"bytes"
	"context"
	"fmt"
	"gabrielsy/imgnow/internal/app"
	fileRepo "gabrielsy/imgnow/internal/repository/file"
	"gabrielsy/imgnow/internal/util"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nfnt/resize"
)

type FileService struct {
	app *app.Application
}

func NewFileService(app *app.Application) *FileService {
	return &FileService{app: app}
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

func (fs *FileService) UploadToR2(file *multipart.FileHeader, customUrl string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	var body io.Reader = src
	var contentLength int64 = file.Size

	if strings.Contains(contentType, "image/") {
		compressedBody, err := compressImage(src, contentType)
		if err != nil {
			util.LogError(err, "Could not compress image, using original", fs.app)
		} else {
			if int64(compressedBody.Len()) < file.Size {
				body = compressedBody
				contentLength = int64(compressedBody.Len())
			} else {
				if _, err := src.Seek(0, io.SeekStart); err != nil {
					return fmt.Errorf("failed to seek original file after failed compression: %w", err)
				}
			}
		}
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", os.Getenv("R2_ACCOUNT_ID")),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
					SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
				}, nil
			},
		))),
	)
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(os.Getenv("R2_BUCKET_NAME")),
		Key:           aws.String(customUrl),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(contentLength),
	})

	return err
}

func (fs *FileService) GetFromR2(customUrl string) (string, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", os.Getenv("R2_ACCOUNT_ID")),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
					SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
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
		Bucket: aws.String(os.Getenv("R2_BUCKET_NAME")),
		Key:    aws.String(customUrl),
	})
	if err != nil {
		return "", err
	}

	return presignedUrl.URL, nil
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
