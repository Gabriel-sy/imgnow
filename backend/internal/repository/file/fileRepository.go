package repository

import (
	"gabrielsy/imgnow/internal/app"
	"gabrielsy/imgnow/internal/types"
)

func FindHash(app *app.Application, hash string) (*types.File, error) {
	query := `SELECT * FROM file WHERE hash = $1`

	rows, err := app.DB.Query(query, hash)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var file types.File
		err := rows.Scan(&file.Id, &file.Hash, &file.Path, &file.OriginalName, &file.Size, &file.Type, &file.CreatedAt)
		if err != nil {
			return nil, err
		}
		return &file, nil
	}

	return nil, nil
}

func CreateFile(app *app.Application, file *types.File) error {
	query := `INSERT INTO file (hash, path, original_name, size, type, created_at) VALUES ($1, $2, $3, $4, $5, $6)`

	tx, err := app.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(query, file.Hash, file.Path, file.OriginalName, file.Size, file.Type, file.CreatedAt)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func HashExists(app *app.Application, hash string) (bool, error) {
	file, err := FindHash(app, hash)
	if err != nil {
		return false, err
	}
	return file != nil, nil
}
