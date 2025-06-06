package repository

import (
	"gabrielsy/imgnow/internal/app"
	"gabrielsy/imgnow/internal/types"
)

func FindHash(app *app.Application, customUrl string) (*types.File, error) {
	query := `SELECT * FROM file WHERE custom_url = $1`

	rows, err := app.DB.Query(query, customUrl)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var file types.File
		err := rows.Scan(&file.Id, &file.CustomUrl, &file.Path, &file.OriginalName, &file.Size, &file.Type, &file.CreatedAt)
		if err != nil {
			return nil, err
		}
		return &file, nil
	}

	return nil, nil
}

func CreateFile(app *app.Application, file *types.File) error {
	query := `INSERT INTO file (custom_url, path, original_name, size, type, created_at) VALUES ($1, $2, $3, $4, $5, $6)`

	tx, err := app.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(query, file.CustomUrl, file.Path, file.OriginalName, file.Size, file.Type, file.CreatedAt)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func HashExists(app *app.Application, customUrl string) (bool, error) {
	file, err := FindHash(app, customUrl)
	if err != nil {
		return false, err
	}
	return file != nil, nil
}
