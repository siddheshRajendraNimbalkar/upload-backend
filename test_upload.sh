#!/bin/bash

# Test the upload system with security fixes
echo "Testing upload backend with security fixes..."

# Create a test file
echo "Creating test file..."
dd if=/dev/zero of=testfile.bin bs=1M count=5

# Start server in background
echo "Starting server..."
go run ./cmd/server &
SERVER_PID=$!
sleep 2

# Test upload
echo "Testing upload..."
go run ./cmd/client --file=testfile.bin

# Cleanup
kill $SERVER_PID
rm testfile.bin

echo "Test completed!"