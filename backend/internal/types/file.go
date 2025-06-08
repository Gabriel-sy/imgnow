package types

import "time"

type FileStatus string

const (
	Pending FileStatus = "pending"
	Active  FileStatus = "active"
	Error   FileStatus = "error"
)

type VideoMessage struct {
	Filename  string `json:"filename"`
	Content   []byte `json:"content"`
	RequestID string `json:"request_id"`
}

type File struct {
	Id           int
	CustomUrl    string
	Path         *string
	OriginalName string
	Size         int
	Type         string
	CreatedAt    time.Time
	Status       FileStatus
}
