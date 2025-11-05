package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"upload-backend/internal/server"
	pb "upload-backend/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type cfg struct {
	GRPCPort    string
	PostgresDSN string
	RedisAddr   string
	JWTSecret   string
	TLSCert     string
	TLSKey      string
	StorageDir  string
}

func mustEnv(k string, optional bool) string {
	v := os.Getenv(k)
	if v == "" && !optional {
		log.Fatalf("missing required env %s", k)
	}
	return v
}

func defaultIfEmpty(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func loadCfg() cfg {
	return cfg{
		GRPCPort:    defaultIfEmpty(os.Getenv("GRPC_PORT"), "50051"),
		PostgresDSN: mustEnv("POSTGRES_DSN", false),
		RedisAddr:   defaultIfEmpty(os.Getenv("REDIS_ADDR"), "localhost:6379"),
		JWTSecret:   mustEnv("JWT_SECRET", os.Getenv("ALLOW_INSECURE") == "true"),
		TLSCert:     os.Getenv("TLS_CERT"),
		TLSKey:      os.Getenv("TLS_KEY"),
		StorageDir:  defaultIfEmpty(os.Getenv("STORAGE_DIR"), "./storage"),
	}
}

func main() {
	// Load and validate configuration
	config := loadCfg()
	
	grpcPort, err := strconv.Atoi(config.GRPCPort)
	if err != nil {
		log.Fatalf("invalid GRPC_PORT: %v", err)
	}

	// Connect to PostgreSQL
	db, err := server.NewUploadDB(config.PostgresDSN)
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}
	fmt.Println("✅ Connected to PostgreSQL")

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
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

	// Configure TLS if certificates are provided
	var grpcServer *grpc.Server
	if config.TLSCert != "" {
		creds, err := credentials.NewServerTLSFromFile(config.TLSCert, config.TLSKey)
		if err != nil {
			log.Fatalf("❌ Failed to load TLS credentials: %v", err)
		}
		grpcServer = grpc.NewServer(grpc.Creds(creds))
		fmt.Println("✅ TLS enabled")
	} else {
		grpcServer = grpc.NewServer()
		fmt.Println("⚠️  Running without TLS")
	}

	// Initialize the upload service
	uploadService := server.NewUploadService(config.RedisAddr, config.StorageDir, db)
	pb.RegisterFileUploadServiceServer(grpcServer, uploadService)

	fmt.Printf("🚀 gRPC server running on port %d\n", grpcPort)

	// Serve gRPC
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Failed to serve gRPC: %v", err)
	}

}
