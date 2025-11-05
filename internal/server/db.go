package server

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UploadRecord represents a single upload record from the DB
type UploadRecord struct {
	FileID     string
	UserID     string
	FileName   string
	StoredPath string
	Status     string
}

type UploadDB struct {
	pool *pgxpool.Pool
}

// NewUploadDB creates a new PostgreSQL connection pool
func NewUploadDB(connStr string) (*UploadDB, error) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	return &UploadDB{pool: pool}, nil
}

// CreateUpload inserts a new upload entry
func (db *UploadDB) CreateUpload(fileID, userID, fileName string, totalChunks int64) error {
	_, err := db.pool.Exec(context.Background(),
		`INSERT INTO uploads(file_id, user_id, file_name, total_chunks, status)
		 VALUES($1, $2, $3, $4, 'in_progress')`,
		fileID, userID, fileName, totalChunks,
	)
	return err
}

// CompleteUpload updates the upload record when merge is done
func (db *UploadDB) CompleteUpload(fileID, storedPath string) error {
	_, err := db.pool.Exec(context.Background(),
		`UPDATE uploads SET status='completed', stored_path=$1 WHERE file_id=$2`,
		storedPath, fileID,
	)
	return err
}

// GetUploadByID retrieves a file upload record by its ID
func (db *UploadDB) GetUploadByID(fileID string) (*UploadRecord, error) {
	var rec UploadRecord
	query := `SELECT file_id, user_id, file_name, stored_path, status FROM uploads WHERE file_id = $1`
	err := db.pool.QueryRow(context.Background(), query, fileID).Scan(
		&rec.FileID, &rec.UserID, &rec.FileName, &rec.StoredPath, &rec.Status,
	)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// DeleteUpload removes an upload record from the database
func (db *UploadDB) DeleteUpload(fileID string) error {
	_, err := db.pool.Exec(context.Background(),
		`DELETE FROM uploads WHERE file_id=$1`,
		fileID,
	)
	return err
}
