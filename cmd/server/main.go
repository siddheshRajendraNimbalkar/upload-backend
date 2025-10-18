package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/redis/go-redis/v9"
	"github.com/siddheshRajendraNimbalkar/upload-backend/internal/server"
	"github.com/siddheshRajendraNimbalkar/upload-backend/pb"
	"google.golang.org/grpc"
)

func main() {
	// -------------------------------
	// Configuration
	// -------------------------------
	grpcPort := 50051
	redisAddr := "localhost:6379"
	tempDir := "storage"
	dbConnStr := "postgresql://upload:upload123@localhost:5432/upload_db?sslmode=disable"

	// -------------------------------
	// Connect to PostgreSQL
	// -------------------------------
	db, err := server.NewUploadDB(dbConnStr)
	if err != nil {
		panic(err)
	}
	fmt.Println("✅ Connected to PostgreSQL")

	// -------------------------------
	// Connect to Redis
	// -------------------------------
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	fmt.Println("✅ Connected to Redis")

	// -------------------------------
	// Start gRPC Server
	// -------------------------------
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Initialize the upload service
	uploadService := server.NewUploadService(redisAddr, tempDir, db)
	pb.RegisterFileUploadServiceServer(grpcServer, uploadService)

	fmt.Printf("🚀 gRPC server running on port %d\n", grpcPort)

	// Serve gRPC
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Failed to serve gRPC: %v", err)
	}
}
