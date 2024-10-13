# Build the application from source
FROM golang:1.22 AS build

WORKDIR /app

COPY ./govnilo/go.mod ./govnilo/go.sum ./
RUN go mod download

COPY ./govnilo/ ./govnilo/

RUN CGO_ENABLED=0 GOOS=linux go -C ./govnilo build -o /checker ./cmd/checker

# Deploy the application binary into a clean image
FROM alpine 

WORKDIR /

COPY --from=build /checker /checker

# USER nonroot:nonroot

ENTRYPOINT ["/checker", "run", "--config", "/config.yaml"]
