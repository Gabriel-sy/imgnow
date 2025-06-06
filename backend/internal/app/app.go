package app

import (
	"database/sql"
	"fmt"
	"gabrielsy/imgnow/internal/repository"
	"log"
	"os"
)

type Application struct {
	Logger *log.Logger
	DB     *sql.DB
}

func NewApplication() (*Application, error) {
	db, err := repository.OpenDB()
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %w", err)
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	app := &Application{
		Logger: logger,
		DB:     db,
	}

	return app, nil
}
