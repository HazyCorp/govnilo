.PHONY: all

build:
	mkdir -p bin
	go build -C ./govnilo -o ../bin/checker ./cmd/govnilo/

protoc:
	protoc -I=./proto --go_out=./pkg/pb --go_opt=paths=source_relative --go-grpc_out=./pkg/pb --go-grpc_opt=paths=source_relative checker.proto

tidy:
	go mod tidy -v

clean:
	rm -rf bin

run:
	docker compose --progress=plain -f ./dev_infra/docker-compose.yml up -d 

build:
	docker compose --progress=plain -f ./dev_infra/docker-compose.yml build

stop:
	docker compose -f ./dev_infra/docker-compose.yml down

logs:
	docker compose -f ./dev_infra/docker-compose.yml logs $(ARGS)
