.PHONY: all

build:
	mkdir -p bin
	go build -C ./src -o ../bin/checker ./cmd/checker

protoc:
	protoc -I=./proto --go_out=./src/pkg/pb --go_opt=paths=source_relative --go-grpc_out=./src/pkg/pb --go-grpc_opt=paths=source_relative checker.proto

tidy:
	go -C ./src/ mod tidy -v

clean:
	rm -rf bin
