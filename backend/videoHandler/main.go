package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"

	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoMessage struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
}

type logger struct {
	logger *log.Logger
}

func (l *logger) logError(err error, msg string) {
	if err != nil {
		l.logger.Printf("%s: %v", msg, err)
	}
}

func main() {
	l := &logger{logger: log.New(os.Stdout, "", log.Ldate|log.Ltime)}

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	l.logError(err, "Failed to connect to RabbitMQ")
	if err != nil {
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	l.logError(err, "Failed to open a channel")
	if err != nil {
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"video_compress_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	l.logError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	l.logError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var videoMsg VideoMessage
			if err := json.Unmarshal(d.Body, &videoMsg); err != nil {
				l.logger.Printf("Error parsing JSON: %v", err)
				d.Ack(false)
				continue
			}

			var outputBuf bytes.Buffer
			var errBuf bytes.Buffer

			cmd := exec.Command("ffmpeg",
				"-i", "pipe:0",
				"-c:v", "libx265",
				"-preset", "medium",
				"-crf", "28",
				"-tag:v", "hvc1",
				"-c:a", "aac",
				"-b:a", "128k",
				"-f", "mp4",
				"-movflags", "+frag_keyframe+empty_moov",
				"pipe:1",
			)

			cmd.Stdin = bytes.NewReader(videoMsg.Content)
			cmd.Stdout = &outputBuf
			cmd.Stderr = &errBuf

			err := cmd.Run()
			if err != nil {
				l.logger.Printf("Error executing FFmpeg: %v", err)
				l.logger.Printf("FFmpeg stderr: %s", errBuf.String())
				d.Ack(false)
				continue
			}

			l.logger.Printf("Video compressed successfully\n")

			returnMessage := VideoMessage{
				Filename: videoMsg.Filename,
				Content:  outputBuf.Bytes(),
			}

			returnMessageBytes, err := json.Marshal(returnMessage)
			if err != nil {
				l.logger.Printf("Error marshalling return message: %v", err)
				d.Ack(false)
				continue
			}

			err = ch.PublishWithContext(context.TODO(),
				"",
				"video_queue",
				false,
				false,
				amqp.Publishing{
					Body:         returnMessageBytes,
					ContentType:  "application/json",
					DeliveryMode: amqp.Persistent,
				},
			)
			if err != nil {
				l.logger.Printf("Error publishing return message: %v", err)
				d.Ack(false)
				continue
			}

			d.Ack(false)
		}
	}()

	l.logger.Printf("Waiting for messages. To exit press CTRL+C")
	<-forever
}
