package server

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/redis/go-redis/v9"
	pb "github.com/siddheshRajendraNimbalkar/upload-backend/pb"
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

// UploadFile handles streaming upload chunks from client
func (s *UploadService) UploadFile(stream pb.FileUploadService_UploadFileServer) error {
	ctx := context.Background()

	// Receive first chunk with metadata
	firstChunk, err := stream.Recv()
	if err == io.EOF {
		return status.Error(codes.InvalidArgument, "no upload data")
	}
	if err != nil {
		return status.Errorf(codes.Internal, "stream recv: %v", err)
	}

	fileID := firstChunk.FileId
	userID := firstChunk.UserId
	fileName := firstChunk.FileName
	totalChunks := firstChunk.TotalChunks

	// Initialize upload metadata in DB
	if err := s.db.CreateUpload(fileID, userID, fileName, totalChunks); err != nil {
		return status.Errorf(codes.Internal, "db insert error: %v", err)
	}

	uploadDir := filepath.Join(s.tempDir, fileID)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return status.Errorf(codes.Internal, "failed to create temp dir: %v", err)
	}

	// Save first chunk
	if err := saveChunk(ctx, s.rdb, uploadDir, fileID, firstChunk); err != nil {
		return err
	}

	// Receive remaining chunks
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "recv chunk error: %v", err)
		}

		if err := saveChunk(ctx, s.rdb, uploadDir, fileID, chunk); err != nil {
			return err
		}
	}

	// Merge chunks
	mergedPath, err := s.mergeChunks(fileID, fileName)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to merge chunks: %v", err)
	}

	// Mark upload completed in DB
	if err := s.db.CompleteUpload(fileID, mergedPath); err != nil {
		return status.Errorf(codes.Internal, "failed to update upload status: %v", err)
	}

	// Return success
	return stream.SendAndClose(&pb.UploadStatus{
		Success:    true,
		Message:    "upload saved",
		StoredPath: mergedPath,
	})
}

// saveChunk saves a chunk to disk and marks it in Redis
func saveChunk(ctx context.Context, rdb *redis.Client, uploadDir, fileID string, chunk *pb.FileChunk) error {
	chunkPath := filepath.Join(uploadDir, fmt.Sprintf("chunk_%d", chunk.ChunkIndex))
	if err := os.WriteFile(chunkPath, chunk.Content, 0644); err != nil {
		return status.Errorf(codes.Internal, "write chunk error: %v", err)
	}

	if err := rdb.Set(ctx, fmt.Sprintf("upload:%s:chunk:%d", fileID, chunk.ChunkIndex), true, 0).Err(); err != nil {
		return status.Errorf(codes.Internal, "redis set error: %v", err)
	}

	fmt.Printf("✅ Received chunk %d for file %s\n", chunk.ChunkIndex, fileID)
	return nil
}

// GetUploadedChunks returns list of uploaded chunk indices from Redis
func (s *UploadService) GetUploadedChunks(ctx context.Context, req *pb.GetChunksRequest) (*pb.GetChunksResponse, error) {
	pattern := fmt.Sprintf("upload:%s:chunk:*", req.FileId)
	keys, err := s.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "redis error: %v", err)
	}

	var chunks []int64
	for _, key := range keys {
		numStr := key[len(fmt.Sprintf("upload:%s:chunk:", req.FileId)):]
		num, err := strconv.ParseInt(numStr, 10, 64)
		if err == nil {
			chunks = append(chunks, num)
		}
	}

	return &pb.GetChunksResponse{
		UploadedChunks: chunks,
	}, nil
}

// mergeChunks joins all chunk files into the final file
func (s *UploadService) mergeChunks(fileID, fileName string) (string, error) {
	dir := filepath.Join(s.tempDir, fileID)
	finalDir := filepath.Join(s.tempDir, "files")
	if err := os.MkdirAll(finalDir, os.ModePerm); err != nil {
		return "", err
	}

	finalPath := filepath.Join(finalDir, fmt.Sprintf("%s_%s", fileID, fileName))
	out, err := os.Create(finalPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(files); i++ {
		chunkPath := filepath.Join(dir, fmt.Sprintf("chunk_%d", i))
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			return "", err
		}
		if _, err := out.Write(data); err != nil {
			return "", err
		}
	}

	fmt.Printf("🎉 Merged file saved at: %s\n", finalPath)
	return finalPath, nil
}
