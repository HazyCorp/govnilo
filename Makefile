.PHONY: all

build:
	mkdir -p bin
	go build -o ./bin/checker ./cmd/checker

protoc:
	protoc -I=./proto --go_out=./pkg/pb --go_opt=paths=source_relative --go-grpc_out=./pkg/pb --go-grpc_opt=paths=source_relative checker.proto

tidy:
	go mod tidy -v

clean:
	rm -rf bin
