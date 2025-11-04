package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	pb "upload-backend/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UploadService implements the gRPC server
type UploadService struct {
	pb.UnimplementedFileUploadServiceServer
	rdb     *redis.Client
	db      *UploadDB
	tempDir string
}

// NewUploadService creates a new UploadService
func NewUploadService(redisAddr, tempDir string, db *UploadDB) *UploadService {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &UploadService{rdb: rdb, db: db, tempDir: tempDir}
}

// InitUpload generates server-owned file ID and initializes upload
func (s *UploadService) InitUpload(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	id := uuid.NewString()
	safe := sanitizeFilename(filepath.Base(req.FileName))
	if err := s.db.CreateUpload(id, req.UserId, safe, req.TotalChunks); err != nil {
		log.Printf("InitUpload error: user_id=%s, file_id=%s, error=%v", req.UserId, id, err)
		return nil, status.Errorf(codes.Internal, "db insert error: %v", err)
	}

	log.Printf("InitUpload success: user_id=%s, file_id=%s, file_name=%s, total_chunks=%d", req.UserId, id, safe, req.TotalChunks)
	return &pb.InitResponse{FileId: id}, nil
}

// UploadFile handles streaming upload chunks from client
func (s *UploadService) UploadFile(stream pb.FileUploadService_UploadFileServer) error {
	ctx := stream.Context()

	// Receive first chunk with metadata
	firstChunk, err := stream.Recv()
	if err == io.EOF {
		return status.Error(codes.InvalidArgument, "no upload data")
	}
	if err != nil {
		return status.Errorf(codes.Internal, "stream recv: %v", err)
	}

	fileID := firstChunk.FileId
	totalChunks := firstChunk.TotalChunks

	// Validate file exists in DB (server must own the ID)
	rec, err := s.db.GetUploadByID(fileID)
	if err != nil {
		log.Printf("UploadFile invalid file_id: file_id=%s, error=%v", fileID, err)
		return status.Errorf(codes.NotFound, "invalid file_id: %v", err)
	}

	tmpDir, _, _ := paths(s.tempDir, fileID, rec.FileName)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return status.Errorf(codes.Internal, "failed to create temp dir: %v", err)
	}

	// Save first chunk
	if err := s.saveChunk(ctx, tmpDir, fileID, firstChunk, totalChunks); err != nil {
		return err
	}

	// Receive remaining chunks
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "recv chunk error: %v", err)
		}

		if err := s.saveChunk(ctx, tmpDir, fileID, chunk, totalChunks); err != nil {
			return err
		}
	}

	// Merge chunks
	mergedPath, err := s.mergeChunks(fileID, rec.FileName, totalChunks)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to merge chunks: %v", err)
	}

	// Mark upload completed in DB
	if err := s.db.CompleteUpload(fileID, mergedPath); err != nil {
		return status.Errorf(codes.Internal, "failed to update upload status: %v", err)
	}

	// Cleanup Redis and temp files
	cleanupChunks(ctx, s.rdb, fileID)
	os.RemoveAll(tmpDir)

	log.Printf("UploadFile success: file_id=%s, stored_path=%s", fileID, mergedPath)

	// Return success
	return stream.SendAndClose(&pb.UploadStatus{
		Success:    true,
		Message:    "upload saved",
		StoredPath: mergedPath,
	})
}

// saveChunk saves a chunk to disk and marks it in Redis with validation
func (s *UploadService) saveChunk(ctx context.Context, tmpDir, fileID string, chunk *pb.FileChunk, totalChunks int64) error {
	// Validate chunk index
	if chunk.ChunkIndex < 0 || chunk.ChunkIndex >= totalChunks {
		return status.Errorf(codes.InvalidArgument, "invalid chunk index %d", chunk.ChunkIndex)
	}

	// Check if chunk already exists (idempotency)
	set, err := listedChunks(ctx, s.rdb, fileID)
	if err == nil {
		if _, exists := set[chunk.ChunkIndex]; exists {
			return nil // Already uploaded, skip
		}
	}

	chunkPath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%d", chunk.ChunkIndex))
	if err := os.WriteFile(chunkPath, chunk.Content, 0644); err != nil {
		return status.Errorf(codes.Internal, "write chunk error: %v", err)
	}

	if err := markChunk(ctx, s.rdb, fileID, chunk.ChunkIndex); err != nil {
		return status.Errorf(codes.Internal, "redis set error: %v", err)
	}

	return nil
}

// GetUploadedChunks returns list of uploaded chunk indices from Redis
func (s *UploadService) GetUploadedChunks(ctx context.Context, req *pb.GetChunksRequest) (*pb.GetChunksResponse, error) {
	set, err := listedChunks(ctx, s.rdb, req.FileId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "redis error: %v", err)
	}

	var chunks []int64
	for chunk := range set {
		chunks = append(chunks, chunk)
	}

	return &pb.GetChunksResponse{
		UploadedChunks: chunks,
	}, nil
}

// mergeChunks joins all chunk files into the final file with validation
func (s *UploadService) mergeChunks(fileID, fileName string, totalChunks int64) (string, error) {
	tmpDir, finalPath, tempFinal := paths(s.tempDir, fileID, fileName)

	// Ensure all chunks exist before merging
	for i := int64(0); i < totalChunks; i++ {
		p := filepath.Join(tmpDir, fmt.Sprintf("chunk_%d", i))
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("missing chunk %d", i)
		}
	}

	// Create final directory
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return "", err
	}

	// Write to temp file first (atomic)
	out, err := os.Create(tempFinal)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Merge chunks in order
	for i := int64(0); i < totalChunks; i++ {
		f, err := os.Open(filepath.Join(tmpDir, fmt.Sprintf("chunk_%d", i)))
		if err != nil {
			return "", err
		}
		io.Copy(out, f)
		f.Close()
	}

	out.Close()
	// Atomic rename
	if err := os.Rename(tempFinal, finalPath); err != nil {
		return "", err
	}

	return finalPath, nil
}

func (s *UploadService) DownloadFile(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadResponse, error) {
	fileID := req.FileId
	var filePath, fileName string

	// Query metadata from DB
	err := s.db.pool.QueryRow(ctx, `SELECT stored_path, file_name FROM uploads WHERE file_id=$1`, fileID).Scan(&filePath, &fileName)
	if err != nil {
		log.Printf("DownloadFile not found: file_id=%s, error=%v", fileID, err)
		return nil, status.Errorf(codes.NotFound, "file not found: %v", err)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("DownloadFile read error: file_id=%s, error=%v", fileID, err)
		return nil, status.Errorf(codes.Internal, "failed to read file: %v", err)
	}

	log.Printf("DownloadFile success: file_id=%s, size=%d", fileID, len(data))
	return &pb.DownloadResponse{
		Content:  data,
		FileName: fileName,
	}, nil
}

// GetUploadMetadata returns file metadata from PostgreSQL and uploaded chunk indices from Redis
func (s *UploadService) GetUploadMetadata(ctx context.Context, req *pb.GetMetadataRequest) (*pb.UploadMetadata, error) {
	// Fetch DB record
	rec, err := s.db.GetUploadByID(req.FileId)
	if err != nil {
		log.Printf("GetUploadMetadata not found: file_id=%s, error=%v", req.FileId, err)
		return nil, status.Errorf(codes.NotFound, "upload not found: %v", err)
	}

	// Determine size
	var size int64
	if rec.Status == "completed" && rec.StoredPath != "" {
		if fi, err := os.Stat(rec.StoredPath); err == nil {
			size = fi.Size()
		}
	} else {
		// Sum sizes of chunk files if present
		tmpDir, _, _ := paths(s.tempDir, rec.FileID, rec.FileName)
		entries, err := os.ReadDir(tmpDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				fp := filepath.Join(tmpDir, entry.Name())
				if fi, err := os.Stat(fp); err == nil {
					size += fi.Size()
				}
			}
		}
	}

	// Get uploaded chunks from Redis using sets
	set, err := listedChunks(ctx, s.rdb, req.FileId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "redis error: %v", err)
	}
	var chunks []int64
	for chunk := range set {
		chunks = append(chunks, chunk)
	}

	return &pb.UploadMetadata{
		FileId:         rec.FileID,
		FileName:       rec.FileName,
		Size:           size,
		UploadedChunks: chunks,
		Status:         rec.Status,
	}, nil
}
