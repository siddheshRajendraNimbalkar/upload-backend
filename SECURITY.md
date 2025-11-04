# Security Improvements

## High-Impact Fixes Implemented

### 1. Server-Owned File IDs
- **Problem**: Client-controlled file IDs could cause collisions/overwrites
- **Solution**: Added `InitUpload` RPC that returns server-generated UUID
- **Impact**: Prevents malicious clients from targeting specific file IDs

### 2. Path Sanitization
- **Problem**: Path traversal attacks via malicious filenames
- **Solution**: `filepath.Base()` + character sanitization in `sanitizeFilename()`
- **Impact**: All files stored safely under controlled directories

### 3. Redis Performance & Security
- **Problem**: `KEYS` command is O(N) and can stall Redis
- **Solution**: Redis Sets with `SADD`/`SMEMBERS` + 24h TTL
- **Impact**: O(1) operations, automatic cleanup, no Redis blocking

### 4. Atomic Operations
- **Problem**: Partial file corruption during merge
- **Solution**: Index-driven merge with gap detection + atomic rename
- **Impact**: Either complete success or clean failure, no partial files

### 5. Input Validation
- **Problem**: Invalid chunk indices could crash server
- **Solution**: Bounds checking: `index >= 0 && index < totalChunks`
- **Impact**: Prevents crashes and resource exhaustion

### 6. Transport Security
- **Problem**: Unencrypted gRPC traffic
- **Solution**: Optional TLS via `TLS_CERT`/`TLS_KEY` environment variables
- **Impact**: Encrypted communication when certificates provided

## Performance Improvements

- **Chunk Size**: 2MB (vs 1KB) for 2000x fewer round trips
- **Idempotency**: Skip duplicate chunks automatically
- **Context Handling**: Proper cancellation in long-running streams
- **File Permissions**: Secure 0755/0644 permissions

## Database Security

- **Constraints**: `status IN ('in_progress','completed','failed')`
- **Indices**: Optimized queries with `(user_id, created_at DESC)`
- **Schema**: Added size_bytes, mime_type, sha256 columns for integrity