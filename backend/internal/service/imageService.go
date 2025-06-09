package service

import (
	"bytes"
	"fmt"
	"gabrielsy/imgnow/internal/app"
	"gabrielsy/imgnow/internal/util"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"strings"

	"github.com/nfnt/resize"
)

type ImageService struct {
	app *app.Application
}

func NewImageService(app *app.Application) *ImageService {
	return &ImageService{app: app}
}

func (is *ImageService) CompressImage(src multipart.File, contentType string) (*bytes.Buffer, error) {
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

func (is *ImageService) HandleImageCompression(file *multipart.FileHeader, contentType string) (io.Reader, int64, error) {
	src, err := file.Open()
	if err != nil {
		return nil, 0, err
	}
	defer src.Close()

	var body io.Reader = src
	var contentLength int64 = file.Size

	compressedBody, err := is.CompressImage(src, contentType)
	if err != nil {
		util.LogError(err, "Could not compress image, using original", is.app)
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
