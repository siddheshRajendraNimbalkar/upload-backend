package server

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UploadDB struct {
	pool *pgxpool.Pool
}

func NewUploadDB(connStr string) (*UploadDB, error) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	return &UploadDB{pool: pool}, nil
}

func (db *UploadDB) CreateUpload(fileID, userID, fileName string, totalChunks int64) error {
	_, err := db.pool.Exec(context.Background(),
		`INSERT INTO uploads(file_id, user_id, file_name, total_chunks, status) VALUES($1,$2,$3,$4,'in_progress')`,
		fileID, userID, fileName, totalChunks,
	)
	return err
}

func (db *UploadDB) CompleteUpload(fileID, storedPath string) error {
	_, err := db.pool.Exec(context.Background(),
		`UPDATE uploads SET status='completed', stored_path=$1 WHERE file_id=$2`,
		storedPath, fileID,
	)
	return err
}
