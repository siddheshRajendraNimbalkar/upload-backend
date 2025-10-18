# upload-grpc-go


## prerequisites
- Go 1.21+
- protoc (Protocol Buffers compiler)
- protoc-gen-go and protoc-gen-go-grpc installed in PATH


Install protobuf plugins:


go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest


Ensure $GOBIN is in your PATH.

## generate pb and run


make proto


Run server (default :50051):


go run ./cmd/server


Run client (upload a file):


go run ./cmd/client --file=/path/to/file


Files are stored under `./storage/files` and temporary parts under `./storage/tmp`.# upload-backend
