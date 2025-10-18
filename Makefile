.PHONY: proto server client


PROTOC_GEN_GO := $(shell which protoc-gen-go)
PROTOC_GEN_GO_GRPC := $(shell which protoc-gen-go-grpc)
PROTO_DIR := proto
OUT_DIR := pb

proto:
	protoc --go_out=$(OUT_DIR) --go-grpc_out=$(OUT_DIR) $(PROTO_DIR)/fileupload.proto



server: proto
go run ./cmd/server


client: proto
go run ./cmd/client --file=path/to/local/file