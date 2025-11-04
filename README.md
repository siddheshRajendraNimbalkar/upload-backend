## Upload Backend (gRPC + gRPC-Gateway)

Chunked file uploads with Redis for chunk tracking, PostgreSQL for metadata, and a REST gateway that exposes select gRPC methods.

### Architecture
- gRPC server: handles streaming uploads, metadata, and download
- Redis: marks uploaded chunk indexes per file during upload
- PostgreSQL: stores upload records (file_id, file_name, status, stored_path, total_chunks)
- Storage: merged files saved under `./storage/files`, in-progress chunks under `./storage/<file_id>`
- gRPC-Gateway: exposes REST for download and metadata

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

- Upload via gRPC streaming (example client):

```
go run ./cmd/client --file=/absolute/path/to/file
```

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

Adjust as needed for your environment.

### Notes
- During upload, chunk presence is tracked in Redis under keys `upload:{file_id}:chunk:{index}`
- On completion, chunks are merged into `./storage/files/{file_id}_{file_name}` and DB status is set to `completed`
- The REST gateway encodes 64-bit integers as strings to be safe for JavaScript clients
