## Upload Backend (gRPC + gRPC-Gateway)

Secure chunked file uploads with Redis for chunk tracking, PostgreSQL for metadata, and a REST gateway that exposes select gRPC methods.

### Architecture
- gRPC server: handles streaming uploads, metadata, and download with server-generated file IDs
- Redis: tracks uploaded chunk indexes using Sets (not KEYS) with TTL
- PostgreSQL: stores upload records with constraints and indices
- Storage: merged files saved under `./storage/files` with sanitized names, temp chunks under `./storage/tmp/<file_id>`
- gRPC-Gateway: exposes REST for download and metadata
- Security: TLS support, path sanitization, atomic operations

### Requirements
- Go 1.21+
- PostgreSQL running locally (see connection string in `cmd/server/main.go`)
- Redis running locally (default `localhost:6379`)
- Protocol Buffers toolchain (protoc) and plugins

Install protobuf plugins:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
```

Ensure `$GOBIN` is on your `PATH`.

### Database

Create the `uploads` table (see `create_uploads_table.sql`):

```
CREATE TABLE IF NOT EXISTS uploads (
    file_id UUID PRIMARY KEY,
    user_id UUID,
    file_name TEXT NOT NULL,
    total_chunks BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'in_progress',
    stored_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Connection string is configured in `cmd/server/main.go` (edit as needed):

```
postgresql://upload:upload123@localhost:5432/upload_db?sslmode=disable
```

### Generate protobufs

Using Makefile (if available):

```
make proto
```

Or directly:

```
protoc -I proto -I third_party/googleapis -I . \
  --go_out=paths=source_relative:. \
  --go-grpc_out=paths=source_relative:. \
  --grpc-gateway_out=paths=source_relative:. \
  proto/fileupload.proto
```

### Run

Start gRPC server (default port 50051):

```
go run ./cmd/server
```

Start REST gateway (default port 8080):

```
go run ./cmd/gateway
```

### Usage

**Upload Process (2-step):**

1. Initialize upload (gets server-generated file ID):
```
rpc InitUpload(InitRequest) returns (InitResponse)
```

2. Upload via gRPC streaming (example client):
```
go run ./cmd/client --file=/absolute/path/to/file
```

**Client now uses 2MB chunks (vs 1KB) for better performance.**

- Download file (REST):

```
curl http://localhost:8080/v1/files/{file_id}
```

Response:

```
{
  "content": "<base64>",
  "fileName": "<original_name>"
}
```

- Get upload metadata (REST):

```
curl http://localhost:8080/v1/uploads/{file_id}/metadata
```

Response (note: int64 values are JSON strings):

```
{
  "fileId": "...",
  "fileName": "...",
  "size": "<int64>",
  "uploadedChunks": ["0", "1", ...],
  "status": "in_progress|completed"
}
```

- Get uploaded chunk indexes (gRPC): `GetUploadedChunks(GetChunksRequest)`

### Configuration

Defaults are set in `cmd/server/main.go`:
- Redis: `localhost:6379`
- Temp/storage dir: `./storage`
- gRPC: `:50051`
- Postgres DSN: see above
- TLS: Set `TLS_CERT` and `TLS_KEY` environment variables to enable

Adjust as needed for your environment.

### Security Features
- **Server-owned file IDs**: InitUpload RPC generates UUIDs server-side
- **Path sanitization**: Filenames cleaned with `filepath.Base()` and dangerous chars removed
- **Redis Sets**: Uses `SADD`/`SMEMBERS` instead of `KEYS` for O(1) operations with 24h TTL
- **Atomic merging**: Index-driven merge with gap detection and atomic rename
- **Validation**: Chunk bounds checking and idempotency
- **TLS support**: Optional via environment variables
- **File permissions**: 0755 for dirs, 0644 for files

### Notes
- During upload, chunk presence is tracked in Redis Sets: `upload:{file_id}:chunks`
- On completion, chunks are merged into `./storage/files/{file_id}_{sanitized_name}` atomically
- Temp files and Redis keys are cleaned up automatically
- The REST gateway encodes 64-bit integers as strings to be safe for JavaScript clients
- Database includes constraints: `status IN ('in_progress','completed','failed')`
