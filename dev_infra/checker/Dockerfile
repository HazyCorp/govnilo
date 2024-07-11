# Build the application from source
FROM golang:1.21 AS build

WORKDIR /app

COPY ./src/go.mod ./src/go.sum ./
RUN go mod download

COPY ./src/ ./src/

RUN CGO_ENABLED=0 GOOS=linux go -C ./src build -o /checker ./cmd/checker

# Deploy the application binary into a lean image
FROM alpine 

WORKDIR /

COPY --from=build /checker /checker

EXPOSE 8080

# USER nonroot:nonroot

ENTRYPOINT ["/checker", "-conf", "/config.yaml"]
