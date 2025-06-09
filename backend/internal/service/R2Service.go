package service

import (
	"context"
	"fmt"
	"gabrielsy/imgnow/internal/app"
	"gabrielsy/imgnow/internal/util"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Service struct {
	app *app.Application
}

func NewR2Service(app *app.Application) *R2Service {
	return &R2Service{app: app}
}

func (rs *R2Service) UploadToR2(body io.Reader, contentType string, contentLength int64, customUrl string) error {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", util.GetEnv("R2_ACCOUNT_ID", rs.app)),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     util.GetEnv("R2_ACCESS_KEY_ID", rs.app),
					SecretAccessKey: util.GetEnv("R2_SECRET_ACCESS_KEY", rs.app),
				}, nil
			},
		))),
	)
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(util.GetEnv("R2_BUCKET_NAME", rs.app)),
		Key:           aws.String(customUrl),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(contentLength),
	})

	return err
}

func (rs *R2Service) GetFromR2(customUrl string) (string, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", util.GetEnv("R2_ACCOUNT_ID", rs.app)),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     util.GetEnv("R2_ACCESS_KEY_ID", rs.app),
					SecretAccessKey: util.GetEnv("R2_SECRET_ACCESS_KEY", rs.app),
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
		Bucket: aws.String(util.GetEnv("R2_BUCKET_NAME", rs.app)),
		Key:    aws.String(customUrl),
	})
	if err != nil {
		return "", err
	}

	return presignedUrl.URL, nil
}

func (rs *R2Service) DeleteFromR2(customUrl string) error {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", util.GetEnv("R2_ACCOUNT_ID", rs.app)),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     util.GetEnv("R2_ACCESS_KEY_ID", rs.app),
					SecretAccessKey: util.GetEnv("R2_SECRET_ACCESS_KEY", rs.app),
				}, nil
			},
		))),
	)
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(util.GetEnv("R2_BUCKET_NAME", rs.app)),
		Key:    aws.String(customUrl),
	})

	return err
}
