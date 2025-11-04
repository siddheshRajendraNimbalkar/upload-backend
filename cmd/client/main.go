package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	pb "upload-backend/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	filePath := flag.String("file", "", "path to file")
	serverAddr := flag.String("server", "localhost:50051", "gRPC server address")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Please provide a file path using --file")
		return
	}

	// Connect to gRPC server  
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := pb.NewFileUploadServiceClient(conn)

	// Open file
	file, err := os.Open(*filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	chunkSize := int64(4 * 1024 * 1024) // 4 MB chunks
	totalChunks := (fileInfo.Size() + chunkSize - 1) / chunkSize

	ctx := context.Background()
	// Add JWT token to context
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer your-jwt-token")

	// Initialize upload with server-generated ID
	initResp, err := client.InitUpload(ctx, &pb.InitRequest{
		FileName:    filepath.Base(fileInfo.Name()),
		TotalChunks: totalChunks,
		UserId:      "user-from-jwt", // This will be overridden by JWT
	})
	if err != nil {
		panic(err)
	}
	fileID := initResp.FileId
	fmt.Println("Uploading file with ID:", fileID)

	// Check already uploaded chunks
	resp, err := client.GetUploadedChunks(ctx, &pb.GetChunksRequest{FileId: fileID})
	if err != nil {
		panic(err)
	}

	uploaded := make(map[int64]bool)
	for _, idx := range resp.UploadedChunks {
		uploaded[idx] = true
	}

	stream, err := client.UploadFile(ctx)
	if err != nil {
		panic(err)
	}

	var chunkIndex int64 = 0
	buf := make([]byte, chunkSize)

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		if uploaded[chunkIndex] {
			fmt.Printf("Skipping already uploaded chunk %d\n", chunkIndex)
			chunkIndex++
			continue
		}

		err = stream.Send(&pb.FileChunk{
			FileId:      fileID,
			FileName:    fileInfo.Name(),
			UserId:      uuid.New().String(), // example
			ChunkIndex:  chunkIndex,
			TotalChunks: totalChunks,
			Content:     buf[:n],
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Sent chunk %d\n", chunkIndex)
		chunkIndex++
	}

	// Close stream and receive upload status
	statusResp, err := stream.CloseAndRecv()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Upload completed: %v, stored path: %s\n", statusResp.Success, statusResp.StoredPath)
}
