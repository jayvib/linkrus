
.PHONY: proto
proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/api.proto

ensure-proto-deps:
	@echo "[go get] ensuring protoc packages are available"
	@go get google.golang.org/grpc
	@go get github.com/golang/protobuf/protoc-gen-go
	
