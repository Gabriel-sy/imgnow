package types

import "time"

type File struct {
	Id           int
	CustomUrl    string
	Path         string
	OriginalName string
	Size         int
	Type         string
	CreatedAt    time.Time
}
