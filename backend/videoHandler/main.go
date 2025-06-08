package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"

	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoMessage struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"content"`
}

func logError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %v", msg, err)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	logError(err, "Failed to connect to RabbitMQ")

	defer conn.Close()

	ch, err := conn.Channel()
	logError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"video_compress_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	logError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	logError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var videoMsg VideoMessage
			if err := json.Unmarshal(d.Body, &videoMsg); err != nil {
				log.Printf("Error parsing JSON: %v", err)
				continue
			}

			pr, pw := io.Pipe()

			go func() {
				defer pw.Close()
				pw.Write(videoMsg.Content)
			}()

			cmd := exec.Command("ffmpeg",
				"-i", "pipe:0",
				"-c:v", "libx265",
				"-preset", "medium",
				"-crf", "28",
				"-tag:v", "hvc1",
				"-c:a", "aac",
				"-b:a", "128k",
				"pipe:1",
			)

			cmd.Stdin = pr
			cmd.Stdout = pw

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Fatalf("Error executing FFmpeg: %s\n%s", err, output)
			}
			fmt.Printf("Video compressed successfully\n")

			var returnMessage VideoMessage
			returnMessage.Filename = videoMsg.Filename
			returnMessage.Content = output

			returnMessageBytes, err := json.Marshal(returnMessage)
			if err != nil {
				log.Printf("Error marshalling return message: %v", err)
				continue
			}

			ch.PublishWithContext(context.TODO(),
				"",
				"video_queue",
				false,
				false,
				amqp.Publishing{Body: returnMessageBytes},
			)
		}
	}()

	log.Printf("Waiting for messages. To exit press CTRL+C")
	<-forever
}
