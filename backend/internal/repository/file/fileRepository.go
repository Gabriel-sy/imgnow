package repository

import (
	"gabrielsy/imgnow/internal/app"
	"gabrielsy/imgnow/internal/types"
	"gabrielsy/imgnow/internal/util"
	"time"
)

func FindFileByCustomUrl(app *app.Application, customUrl string) (*types.File, error) {
	query := `SELECT * FROM file WHERE custom_url = $1`

	rows, err := app.DB.Query(query, customUrl)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var file types.File
		err := rows.Scan(
			&file.Id,
			&file.CustomUrl,
			&file.Path,
			&file.OriginalName,
			&file.Size,
			&file.Type,
			&file.CreatedAt,
			&file.Status,
			&file.Vizualizations,
			&file.DeletesAfterDownload,
			&file.DeletedAt,
			&file.DownloadsForDeletion,
			&file.DeletesAfterVizualizations,
			&file.VizualizationsForDeletion,
			&file.LastVizualization,
			&file.ExpiresIn,
			&file.Downloads,
			&file.Password,
		)
		if err != nil {
			return nil, err
		}
		return &file, nil
	}

	return nil, nil
}

func CreateFile(app *app.Application, file *types.File) error {
	query := `INSERT INTO file (custom_url, original_name, size, type, created_at, status) VALUES ($1, $2, $3, $4, $5, $6)`

	tx, err := app.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(query, file.CustomUrl, file.OriginalName, file.Size, file.Type, file.CreatedAt, file.Status)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func CustomUrlExists(app *app.Application, customUrl string) (bool, error) {
	file, err := FindFileByCustomUrl(app, customUrl)
	if err != nil {
		return false, err
	}
	return file != nil, nil
}

func UpdateFileStatus(app *app.Application, customUrl string, status types.FileStatus) error {
	query := `UPDATE file SET status = $1 WHERE custom_url = $2`

	_, err := app.DB.Exec(query, status, customUrl)
	if err != nil {
		return err
	}

	return nil
}

func UpdateFilePath(app *app.Application, customUrl string, path string) error {
	query := `UPDATE file SET path = $1 WHERE custom_url = $2`

	tx, err := app.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(query, path, customUrl)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func IncrementVizualizations(app *app.Application, customUrl string) error {
	query := `UPDATE file 
		SET vizualizations = vizualizations + 1,
			last_vizualization = CURRENT_TIMESTAMP
		WHERE custom_url = $1`

	_, err := app.DB.Exec(query, customUrl)
	return err
}

func MarkFileAsDeleted(app *app.Application, customUrl string) error {
	query := `UPDATE file 
		SET deleted_at = CURRENT_TIMESTAMP,
			status = $1
		WHERE custom_url = $2`

	_, err := app.DB.Exec(query, types.Error, customUrl)
	return err
}

func UpdateDeletionDownloadSettings(app *app.Application, customUrl string, deletesAfterDownload bool, downloadsForDeletion *int) error {
	query := `UPDATE file 
		SET deletes_after_download = $1,
			downloads_for_deletion = $2
		WHERE custom_url = $3`

	_, err := app.DB.Exec(query, deletesAfterDownload, downloadsForDeletion, customUrl)
	return err
}

func UpdateDeletionVizualizationSettings(app *app.Application, customUrl string, deletesAfterVizualizations bool, vizualizationsForDeletion *int) error {
	query := `UPDATE file 
		SET deletes_after_vizualizations = $1,
			vizualizations_for_deletion = $2
		WHERE custom_url = $3`

	_, err := app.DB.Exec(query, deletesAfterVizualizations, vizualizationsForDeletion, customUrl)
	return err
}

func GetFileDeletionInfo(app *app.Application, customUrl string) (*types.File, error) {
	query := `SELECT id, custom_url, vizualizations, deletes_after_download, downloads_for_deletion, 
		deletes_after_vizualizations, vizualizations_for_deletion, deleted_at, downloads
		FROM file WHERE custom_url = $1`

	rows, err := app.DB.Query(query, customUrl)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var file types.File
		err := rows.Scan(
			&file.Id,
			&file.CustomUrl,
			&file.Vizualizations,
			&file.DeletesAfterDownload,
			&file.DownloadsForDeletion,
			&file.DeletesAfterVizualizations,
			&file.VizualizationsForDeletion,
			&file.DeletedAt,
			&file.Downloads,
		)
		if err != nil {
			return nil, err
		}
		return &file, nil
	}

	return nil, nil
}

func UpdateExpirationSettings(app *app.Application, customUrl string, expiresIn *time.Time) error {
	query := `UPDATE file 
		SET expires_in = $1
		WHERE custom_url = $2`

	_, err := app.DB.Exec(query, expiresIn, customUrl)
	return err
}

func GetExpiredFiles(app *app.Application) ([]*types.File, error) {
	query := `SELECT * FROM file 
		WHERE expires_in IS NOT NULL 
		AND expires_in <= CURRENT_TIMESTAMP 
		AND deleted_at IS NULL`

	rows, err := app.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*types.File
	for rows.Next() {
		var file types.File
		err := rows.Scan(
			&file.Id,
			&file.CustomUrl,
			&file.Path,
			&file.OriginalName,
			&file.Size,
			&file.Type,
			&file.CreatedAt,
			&file.Status,
			&file.Vizualizations,
			&file.DeletesAfterDownload,
			&file.DeletedAt,
			&file.DownloadsForDeletion,
			&file.DeletesAfterVizualizations,
			&file.VizualizationsForDeletion,
			&file.LastVizualization,
			&file.ExpiresIn,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}

	return files, nil
}

func IncrementDownloads(app *app.Application, customUrl string) error {
	query := `UPDATE file 
		SET downloads = downloads + 1
		WHERE custom_url = $1`

	_, err := app.DB.Exec(query, customUrl)
	return err
}

func UpdatePassword(app *app.Application, customUrl string, password *string) error {
	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		return err
	}
	query := `UPDATE file 
		SET password = $1
		WHERE custom_url = $2`

	_, err = app.DB.Exec(query, hashedPassword, customUrl)
	return err
}

func GetFilePassword(app *app.Application, customUrl string) (*string, error) {
	query := `SELECT password FROM file WHERE custom_url = $1`

	var hashedPassword *string
	err := app.DB.QueryRow(query, customUrl).Scan(&hashedPassword)
	if err != nil {
		return nil, err
	}
	return hashedPassword, nil
}