package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strconv"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/siddheshRajendraNimbalkar/upload-backend/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	grpcServerEndpoint := flag.String("grpc-server-endpoint", "localhost:50051", "gRPC server endpoint")
	flag.Parse()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := pb.RegisterFileUploadServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)
	if err != nil {
		panic(fmt.Errorf("failed to start gateway: %v", err))
	}

	// Add custom REST upload endpoint
	mux.HandlePath("POST", "/v1/upload", handleUpload)

	// Add CORS middleware
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	fmt.Println("🌍 gRPC-Gateway (REST) server running on port 8080")
	err = http.ListenAndServe(":8080", corsHandler(mux))
	if err != nil {
		panic(fmt.Errorf("failed to start HTTP server: %v", err))
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get other form fields
	fileId := r.FormValue("fileId")
	fileName := r.FormValue("fileName")
	userId := r.FormValue("userId")
	totalChunksStr := r.FormValue("totalChunks")

	if fileId == "" || fileName == "" || userId == "" || totalChunksStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	_, err = strconv.ParseInt(totalChunksStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid totalChunks", http.StatusBadRequest)
		return
	}

	// For now, just return success
	// In a real implementation, you would:
	// 1. Connect to gRPC server
	// 2. Stream the file in chunks
	// 3. Handle the upload process

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "message": "File uploaded successfully", "fileId": "%s"}`, fileId)
}