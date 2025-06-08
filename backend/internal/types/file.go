package types

import "time"

type FileStatus string

const (
	Pending FileStatus = "pending"
	Active  FileStatus = "active"
	Error   FileStatus = "error"
)

type File struct {
	Id           int
	CustomUrl    string
	Path         string
	OriginalName string
	Size         int
	Type         string
	CreatedAt    time.Time
	Status       FileStatus
}
