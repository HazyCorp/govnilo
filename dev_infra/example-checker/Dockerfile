# Build the application from source
FROM golang:1.22 AS build

WORKDIR /app

COPY ./govnilo ./govnilo

COPY ./example/go.mod ./example/go.sum ./example/

RUN go -C ./example/ mod download

COPY ./example/ ./example/

RUN CGO_ENABLED=0 go -C ./example build -o /example ./cmd/main.go

# Deploy the application binary into a clean image
FROM alpine:3.20.3

WORKDIR /

COPY --from=build /example /example

# USER nonroot:nonroot

ENTRYPOINT ["/example", "run"]
