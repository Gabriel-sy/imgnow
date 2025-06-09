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
	Id                         int
	CustomUrl                  string
	Path                       *string
	OriginalName               string
	Size                       int
	Type                       string
	CreatedAt                  time.Time
	Status                     FileStatus
	Vizualizations             int
	Downloads                  int
	DeletesAfterDownload       bool
	DeletedAt                  *time.Time
	DownloadsForDeletion       *int
	DeletesAfterVizualizations bool
	VizualizationsForDeletion  *int
	LastVizualization          *time.Time
	ExpiresIn                  *time.Time
}

type FileSettings struct {
	ExpiresIn                  *time.Time  `json:"expiresIn"`
	DeletesAfterDownload       bool       `json:"deletesAfterDownload"`
	DownloadsForDeletion       *int       `json:"downloadsForDeletion"`
	DeletesAfterVizualizations bool       `json:"deletesAfterVizualizations"`
	VizualizationsForDeletion  *int       `json:"vizualizationsForDeletion"`
	Password                   *string     `json:"password"`
}
