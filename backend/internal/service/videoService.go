package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gabrielsy/imgnow/internal/app"
	"gabrielsy/imgnow/internal/types"
	"gabrielsy/imgnow/internal/util"
	"io"
	"mime/multipart"
	"path/filepath"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoService struct {
	app         *app.Application
	amqpConn    *amqp.Connection
	amqpChannel *amqp.Channel
}

func NewVideoService(app *app.Application) *VideoService {
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
	return &VideoService{app: app, amqpConn: amqpConn, amqpChannel: amqpChannel}
}

func (vs *VideoService) HandleVideoCompression(file *multipart.FileHeader) (io.Reader, int64, error) {
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
	responseQueue, err := vs.setupResponseQueue(requestID)
	if err != nil {
		return nil, 0, err
	}

	// Setup consumer
	msgs, err := vs.setupQueueConsumer(responseQueue.Name)
	if err != nil {
		return nil, 0, err
	}

	// Publish video for compression
	if err := vs.publishVideoForCompression(message); err != nil {
		return nil, 0, err
	}

	// Wait for response with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return vs.waitForCompressedVideo(ctx, msgs, requestID)
}

func (vs *VideoService) setupResponseQueue(requestID string) (*amqp.Queue, error) {
	responseQueue, err := vs.amqpChannel.QueueDeclare(
		"video_queue"+requestID, // unique queue name
		true,                    // durable
		true,                    // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		util.LogError(err, "Failed to declare response queue", vs.app)
		return nil, fmt.Errorf("failed to declare response queue: %w", err)
	}
	return &responseQueue, nil
}

func (vs *VideoService) setupQueueConsumer(queueName string) (<-chan amqp.Delivery, error) {
	msgs, err := vs.amqpChannel.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		util.LogError(err, "Failed to register a consumer", vs.app)
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}
	return msgs, nil
}

func (vs *VideoService) publishVideoForCompression(message types.VideoMessage) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		util.LogError(err, "Failed to marshal message", vs.app)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return vs.amqpChannel.PublishWithContext(context.TODO(),
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

func (vs *VideoService) waitForCompressedVideo(ctx context.Context, msgs <-chan amqp.Delivery, requestID string) (io.Reader, int64, error) {
	for {
		select {
		case msg := <-msgs:
			var response types.VideoMessage
			if err := json.Unmarshal(msg.Body, &response); err != nil {
				util.LogError(err, "Failed to unmarshal response", vs.app)
				return nil, 0, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			if response.RequestID == requestID {
				return bytes.NewReader(response.Content), int64(len(response.Content)), nil
			}
		case <-ctx.Done():
			util.LogError(nil, "Timeout waiting for video compression response", vs.app)
			return nil, 0, fmt.Errorf("timeout waiting for video compression response")
		}
	}
}
