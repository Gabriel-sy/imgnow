package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	RabbitMQURL  string
	ConsumeQueue string
	FFmpegPreset string
	FFmpegCRF    string
	NumWorkers   int
}

type VideoMessage struct {
	Filename  string `json:"filename"`
	Content   []byte `json:"content"`
	RequestID string `json:"request_id"`
}

type logger struct {
	logger *log.Logger
}

func (l *logger) logError(err error, msg string, args ...interface{}) {
	if err != nil {
		l.logger.Printf("ERROR: "+msg+": %v", append(args, err)...)
	}
}

func (l *logger) logInfo(msg string, args ...interface{}) {
	l.logger.Printf("INFO: "+msg, args...)
}

func getEnv(key, fallback string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func loadConfig() Config {
	numWorkersStr := getEnv("NUM_WORKERS", strconv.Itoa(runtime.NumCPU()))
	numWorkers, err := strconv.Atoi(numWorkersStr)
	if err != nil {
		log.Fatalf("Invalid NUM_WORKERS value: %s. Must be an integer.", numWorkersStr)
	}

	return Config{
		RabbitMQURL:  getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		ConsumeQueue: getEnv("CONSUME_QUEUE", "video_compress_queue"),
		FFmpegPreset: getEnv("FFMPEG_PRESET", "medium"),
		FFmpegCRF:    getEnv("FFMPEG_CRF", "28"),
		NumWorkers:   numWorkers,
	}
}

func main() {
	l := &logger{logger: log.New(os.Stdout, "", log.Ldate|log.Ltime)}
	cfg := loadConfig()

	l.logInfo("Configuration loaded: %+v", cfg)

	conn, err := amqp.Dial(cfg.RabbitMQURL)
	l.logError(err, "Failed to connect to RabbitMQ")
	if err != nil {
		os.Exit(1)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	l.logError(err, "Failed to open a channel")
	if err != nil {
		os.Exit(1)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		cfg.ConsumeQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	l.logError(err, "Failed to declare a queue")
	if err != nil {
		os.Exit(1)
	}

	err = ch.Qos(
		1,     // prefetchCount: 1 message at a time
		0,     // prefetchSize
		false, // global
	)
	l.logError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer
		false, // auto-ack: Set to false for manual acknowledgement
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	l.logError(err, "Failed to register a consumer")
	if err != nil {
		os.Exit(1)
	}

	// Start a number of worker goroutines to process messages concurrently.
	l.logInfo("Starting %d workers...", cfg.NumWorkers)
	for i := 1; i <= cfg.NumWorkers; i++ {
		go worker(i, l, cfg, ch, msgs)
	}

	l.logInfo("Waiting for messages. To exit press CTRL+C")

	// --- Graceful Shutdown ---
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownChan

	l.logInfo("Shutdown signal received. Finishing in-progress work.")
	// The channel and connection will be closed by the defer statements.
	l.logInfo("Shutdown complete.")
}

// Params:
// id: The ID of the worker
// l: The logger
// cfg: The .env configuration
// ch: The channel
// deliveries: The deliveries
func worker(id int, l *logger, cfg Config, ch *amqp.Channel, deliveries <-chan amqp.Delivery) {
	l.logInfo("Worker %d started and waiting for messages", id)
	for d := range deliveries {
		l.logInfo("Worker %d received a video to compress", id)
		var videoMsg VideoMessage
		if err := json.Unmarshal(d.Body, &videoMsg); err != nil {
			l.logError(err, "Worker %d: Error parsing JSON", id)
			d.Nack(false, false)
			continue
		}

		outputBuf, err := runFFmpeg(videoMsg.Content, cfg)
		if err != nil {
			l.logError(err, "Worker %d: Error executing FFmpeg", id)
			d.Nack(false, false)
			continue
		}

		l.logInfo("Worker %d: Video compressed successfully", id)

		returnMessage := VideoMessage{
			Filename:  videoMsg.Filename,
			Content:   outputBuf,
			RequestID: videoMsg.RequestID,
		}

		returnMessageBytes, err := json.Marshal(returnMessage)
		if err != nil {
			l.logError(err, "Worker %d: Error marshalling return message", id)
			d.Nack(false, false)
			continue
		}

		replyTo := d.ReplyTo
		if replyTo == "" {
			l.logError(nil, "Worker %d: No ReplyTo queue specified", id)
			d.Nack(false, false)
			continue
		}

		err = ch.PublishWithContext(context.TODO(),
			"",      // exchange
			replyTo, // routing key (the reply queue name)
			false,   // mandatory
			false,   // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Body:         returnMessageBytes,
			},
		)
		if err != nil {
			l.logError(err, "Worker %d: Error publishing return message", id)
			d.Nack(false, true)
			continue
		}

		d.Ack(false)
	}
}

// Params:
// videoContent: The video content
// cfg: The .env configuration
func runFFmpeg(videoContent []byte, cfg Config) ([]byte, error) {
	var outputBuf bytes.Buffer
	var errBuf bytes.Buffer

	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-c:v", "libx265",
		"-preset", cfg.FFmpegPreset,
		"-crf", cfg.FFmpegCRF,
		"-tag:v", "hvc1",
		"-c:a", "aac",
		"-b:a", "128k",
		"-f", "mp4",
		"-movflags", "+frag_keyframe+empty_moov",
		"pipe:1",
	)

	cmd.Stdin = bytes.NewReader(videoContent)
	cmd.Stdout = &outputBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("FFmpeg command failed: %w\nFFmpeg stderr: %s", err, errBuf.String())
	}

	return outputBuf.Bytes(), nil
}
